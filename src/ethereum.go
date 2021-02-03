package incentives

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/videocoin/vidpool/flat"
)

type TransactOpts struct {
	DryRun bool
	Price  *big.Int
}

func (app *App) batchEngine(price *big.Int) (flat.BatchEngine, error) {
	opts := *app.opts
	opts.GasPrice = price
	client, err := ethclient.Dial(app.conf.EthereumURL)
	if err != nil {
		return flat.BatchEngine{}, fmt.Errorf("failed to dial %v: %w", app.conf.EthereumURL, err)
	}
	engine, err := flat.NewBatchEngine(app.log, &opts, client, app.conf.ERC20Address, app.conf.BatchContractAddress)
	if err != nil {
		return engine, err
	}
	// old erc20 standart is using increaseApproval
	engine.AllowanceMethod = app.conf.IncreaseAllowanceMethod
	return engine, nil
}

func (app *App) Transact(ctx context.Context, payments []Payment, opts *TransactOpts) ([]Payment, error) {
	engine, err := app.batchEngine(opts.Price)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize batch engine: %w", err)
	}
	// casting to vidpool data structure
	epayments := []flat.Payment{}
	for i := range payments {
		if (payments[i].Transaction != common.Hash{}) {
			continue
		}
		app.log.Infof("executing payment to %s for %v",
			payments[i].Address.String(), payments[i].Amount)
		epayments = append(epayments, flat.Payment{
			Address: payments[i].Address, Amount: payments[i].Amount})
	}
	if err := engine.Validate(ctx, epayments); err != nil {
		return nil, err
	}
	executed, txs, err := engine.Prepare(ctx, epayments,
		&flat.ExecuteOpts{DryRun: opts.DryRun, BatchSize: app.conf.BatchSize})
	if err != nil {
		return nil, fmt.Errorf("failed to prepare transactions: %w", err)
	}
	if !opts.DryRun {
		for _, tx := range txs {
			if err := engine.Execute(ctx, tx); err != nil {
				return nil, err
			}
		}
	}
	j := 0
	for i := range payments {
		if (payments[i].Transaction != common.Hash{}) {
			continue
		}
		payments[i].Transaction = executed[j].Hash
		j++
	}
	return payments, nil
}
