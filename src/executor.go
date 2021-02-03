package incentives

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/videocoin/vidpool/price"
)

var (
	ErrPending = errors.New("request is pending")
	ErrInvalid = errors.New("request id is invalid")
)

// AsyncExecute unlike Execute will run the job in the background and save result to the results map.
func (app *App) AsyncExecute(ctx context.Context, inputType bool, req *Request) (int, error) {
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
		err := app.Execute(ctx, inputType, req)
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

func (app *App) RunHistoryCleaner(ctx context.Context) {
	ticker := time.NewTicker(app.conf.HistoryRetention)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				app.mu.Lock()
				for id, res := range app.results {
					if now.Sub(res.timestamp) >= app.conf.HistoryRetention {
						app.log.Debugf("result for request with id %d (timestamp %v) was removed from memory", id, res.timestamp)
						delete(app.results, id)
					}
				}
				app.mu.Unlock()
			}
		}
	}()
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

func (app *App) Execute(ctx context.Context, inputTypeCsv bool, req *Request) error {
	var payments []Payment
	var err error
	if inputTypeCsv {
		payments, err = app.ReadPaymentsFile(ctx, req)		
	} else {
		payments, err = app.RetrievePayments(ctx, req)
	}
	if err != nil {
		return err
	}

	client, err := ethclient.Dial(app.conf.EthereumURL)
	if err != nil {
		return fmt.Errorf("failed to dial %v: %w", app.conf.EthereumURL, err)
	}

	gasPrice, err := price.Estimator{Log: app.log, Client: client}.Estimate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to estimate gas price: %w", err)
	}
	app.log.Infof("gas price for next transactions %v", gasPrice)
	payments, err = app.Transact(ctx, payments, &TransactOpts{
		DryRun: app.conf.DryRun,
		Price:  gasPrice,
	})
	if err != nil {
		return err
	}
	if inputTypeCsv {	
		return app.WritePaymentsReceipt(ctx, req, payments)		
	} else {
		return app.UpdateSpreadsheet(ctx, req, payments)
	}
}
