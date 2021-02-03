package incentives

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

const (
	RequestCompleted = "COMPLETED"
	RequestFailed    = "FAILED"
	RequestPending   = "PENDING"
	RequestInvalid   = "INVALID"
)

type Response struct {
	ID      int    `json:"id"`
	State   string `json:"state"`
	Message string `json:"message"`
}

type Request struct {
	SpreadsheetID string `json:"spreadsheetId"`
	// add starting row?
}

func extractTokenFromRequest(req *http.Request) (*oauth2.Token, error) {
	header := req.Header.Get("Authorization")
	if len(header) == 0 {
		return nil, fmt.Errorf("authorization header must be set")
	}
	parts := strings.Split(header, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid header format. expecting header in the format `TYPE TOKEN`")
	}
	return &oauth2.Token{AccessToken: parts[1], TokenType: parts[0]}, nil
}

func executeHandler(app *App) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(&Response{Message: "Use POST."})
			return
		}
		app.log.Debugf("received request %v", req.URL)

		var r Request
		if err := json.NewDecoder(req.Body).Decode(&r); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(rw).Encode(&Response{Message: "Failed to read body."})
			return
		}
		id, err := app.AsyncExecute(req.Context(), false, &r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(&Response{Message: err.Error()})
			return
		}
		rw.WriteHeader(http.StatusOK)
		json.NewEncoder(rw).Encode(&Response{ID: id})
	}
}

func stateHandler(app *App) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(&Response{Message: "Use GET."})
			return
		}

		id, err := strconv.Atoi(mux.Vars(req)["id"])
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(&Response{Message: fmt.Sprintf("%v expected to be a valid integer.", mux.Vars(req)["id"])})
			return
		}
		rw.WriteHeader(http.StatusOK)
		err = app.GetState(id)
		if err == nil {
			json.NewEncoder(rw).Encode(&Response{ID: id, State: RequestCompleted})
		} else if errors.Is(err, ErrPending) {
			json.NewEncoder(rw).Encode(&Response{ID: id, State: RequestPending})
		} else if errors.Is(err, ErrInvalid) {
			json.NewEncoder(rw).Encode(&Response{ID: id, State: RequestInvalid})
		} else {
			json.NewEncoder(rw).Encode(&Response{ID: id, State: RequestFailed, Message: err.Error()})
		}
	}
}

func Serve(ctx context.Context, app *App) error {
	mux := mux.NewRouter()
	mux.HandleFunc("/v1/execute", executeHandler(app))
	mux.HandleFunc("/v1/state/{id}", stateHandler(app))
	server := &http.Server{
		Addr:           app.conf.Address,
		Handler:        mux,
		ReadTimeout:    app.conf.ReadTimeout,
		WriteTimeout:   app.conf.WriteTimeout,
		MaxHeaderBytes: app.conf.MaxHeaderBytes,
	}
	app.RunHistoryCleaner(ctx)
	app.log.Infof("http server is listening on %v", app.conf.Address)
	go func() {
		<-ctx.Done()
		app.log.Info("application received interrupt")
		err := server.Close()
		if err != nil {
			app.log.Debugf("server closed with error %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			app.log.Errorf("http server crashed with %v", err)
			return err
		}
	}
	app.log.Info("server exited")
	return nil
}
