package incentives

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"google.golang.org/api/sheets/v4"
)

type Payment struct {
	ID          int            `json:"id"`
	Address     common.Address `json:"address"`
	Amount      *big.Int       `json:"amount"`
	Transaction common.Hash    `json:"tx"`
}

func (app *App) SheetsService(ctx context.Context) (*sheets.Service, error) {
	return sheets.New(app.oauthConf.Client(ctx))
}

func (app *App) readRange() string {
	return fmt.Sprintf("%s%d:%s", app.conf.WorkerColumn, app.conf.StartRow,
		app.conf.TxColumn)
}

func (app *App) updateRange() string {
	return fmt.Sprintf("%s%d:%s", app.conf.TxColumn, app.conf.StartRow,
		app.conf.TxColumn)
}

func (app *App) RetrievePayments(ctx context.Context, req *Request) ([]Payment, error) {
	srv, err := app.SheetsService(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := srv.Spreadsheets.Values.Get(
		req.SpreadsheetID, app.readRange()).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	payments := make([]Payment, len(resp.Values))
	ether := big.NewInt(params.Ether)
	for i := range payments {
		row := resp.Values[i]
		app.log.Debugf("processing row %v", row)
		if len(row) < 2 {
			return nil, fmt.Errorf("row with idx %d must have two columns field (address and amount)", app.conf.StartRow+i)
		}
		payments[i].ID = i

		address := row[0].(string)
		if len(address) == 0 {
			return nil, fmt.Errorf("`address` at row with idx %d must not be empty", app.conf.StartRow+i)
		}
		payments[i].Address = common.HexToAddress(address)
		amount := row[1].(string)
		if len(amount) == 0 {
			return nil, fmt.Errorf("`amount` at row with idx %d must not be empty", app.conf.StartRow+i)
		}
		payments[i].Amount, _ = new(big.Int).SetString(amount, 0)
		payments[i].Amount = payments[i].Amount.Mul(payments[i].Amount, ether)
		if len(row) > 2 {
			txhash := row[2].(string)
			if len(txhash) > 0 {
				payments[i].Transaction = common.HexToHash(txhash)
			}
		}
	}
	return payments, nil
}

func (app *App) UpdateSpreadsheet(ctx context.Context, req *Request, payments []Payment) error {
	updates := make([][]interface{}, len(payments))
	for i := range payments {
		updates[i] = []interface{}{payments[i].Transaction.String()}
	}
	srv, err := app.SheetsService(ctx)
	if err != nil {
		return err
	}
	_, err = srv.Spreadsheets.Values.
		Update(req.SpreadsheetID, app.updateRange(),
			&sheets.ValueRange{Range: app.updateRange(), Values: updates}).
		ValueInputOption("RAW").
		Context(ctx).Do()
	return err
}
