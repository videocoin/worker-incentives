package incentives

import (
	"context"
	"errors"
)

var (
	ErrPending = errors.New("request is pending")
	ErrInvalid = errors.New("request id is invalid")
)

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
