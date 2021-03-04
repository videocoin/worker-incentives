package incentives

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrPending = errors.New("request is pending")
	ErrInvalid = errors.New("request id is invalid")
)

// AsyncExecute unlike Execute will run the job in the background and save result to the results map.
func (app *App) AsyncExecute(ctx context.Context, inputType bool, inputfile string, outputfile string) (int, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.state == statePending {
		return 0, fmt.Errorf("request with ID %d is in progress. Please wait.", app.id)
	}
	app.id++
	app.state = statePending
	go func() {
		// it is intentionally doesn't inherit method ctx, as it will be tied to http
		// request
		ctx, _ = context.WithTimeout(context.Background(), app.conf.JobTimeout)
		err := app.Execute(ctx, inputfile, outputfile)
		app.mu.Lock()
		defer app.mu.Unlock()
		app.state = stateIdle
		// FIXME this will potentially leak.
		if err != nil {
			app.log.Errorf("execution %d failed with %v", app.id, err)
		}
		app.results[app.id] = &result{timestamp: time.Now(), err: err}
	}()
	return app.id, nil
}


func (app *App) GetState(id int) error {
	app.mu.Lock()
	defer app.mu.Unlock()
	if id > app.id {
		return ErrInvalid
	}
	if app.id == id && app.state == statePending {
		return ErrPending
	}
	result, ok := app.results[id]
	if !ok {
		return ErrInvalid
	}
	return result.err
}

func (app *App) Execute(ctx context.Context, inputfile string, outputfile string) error {
	var payments []Payment
	var err error
	payments, err = app.ReadPaymentsFile(ctx, inputfile)		

	if err != nil {
		return err
	}

	payments, err = app.Transact(ctx, payments)
	if err != nil {
		return err
	}
	return app.WritePaymentsReceipt(ctx, outputfile, payments)		
}
