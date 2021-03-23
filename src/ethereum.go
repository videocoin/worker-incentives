package app

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Transfer(
	ctx context.Context,
	client *ethclient.Client,
	opts *bind.TransactOpts,
	senderPrivKey *ecdsa.PrivateKey,
	receiver common.Address,
	amount *big.Int) (error, common.Hash) {

	var txHash common.Hash
	nonce, err := client.PendingNonceAt(ctx, opts.From)
	if err != nil {
		return err, txHash
	}
	value := amount
	gasPrice := big.NewInt(30000000000)
	gasLimit := uint64(21000)
	var data []byte
	//chainID := big.NewInt(1337)
	tx := types.NewTransaction(nonce, receiver, value, gasLimit, gasPrice, data)
	s := types.HomesteadSigner{}
	signedTx, err := types.SignTx(tx, s, senderPrivKey)
	if err != nil {
		return err, txHash
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err, txHash
	}
	return nil, signedTx.Hash()
}

func (e *Engine) Transact(ctx context.Context, execOpts *ExecuteOpts, payments []Payment) ([]Payment, error) {
	for i := range payments {
		err, txHash := Transfer(ctx, execOpts.client, execOpts.opts, execOpts.senderPrivKey, payments[i].Address, payments[i].Amount)
		if err != nil {
			return payments, err
		}
		payments[i].Transaction = txHash
		i++
	}
	return payments, nil
}
