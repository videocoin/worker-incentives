package app

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
)

type ExecuteOpts struct {
	client     *ethclient.Client
	opts       *bind.TransactOpts
	inputfile  string
	outputfile string
}

type Engine struct {
	log *logrus.Entry
}

var (
	ErrPending = errors.New("request is pending")
	ErrInvalid = errors.New("request id is invalid")
)

func NewEngine(log *logrus.Entry,

) Engine {
	return Engine{
		log: log,
	}
}

func (e *Engine) Execute(ctx context.Context, execOpts *ExecuteOpts) error {
	var payments []Payment
	var err error
	payments, err = ReadPaymentsFile(execOpts.inputfile)

	if err != nil {
		return err
	}

	payments, err = e.Transact(ctx, execOpts, payments)
	if err != nil {
		return err
	}
	return WritePaymentsReceipt(execOpts.outputfile, payments)
}
