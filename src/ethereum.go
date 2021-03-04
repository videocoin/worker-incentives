package incentives

import (
	"context"
	//"fmt"
	"math/big"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/core/types"	
	"github.com/ethereum/go-ethereum/accounts/abi/bind"	
	//"github.com/videocoin/vidpool/flat"
)

func Transfer(client *ethclient.Client, opts *bind.TransactOpts, senderPrivKey *ecdsa.PrivateKey, receiver common.Address, amount *big.Int) (error, common.Hash) {
	var txHash common.Hash
	nonce, err := client.PendingNonceAt(context.Background(), opts.From)
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
	return nil, txHash
}

func (app *App) Transact(ctx context.Context, payments []Payment) ([]Payment, error) {
	for i := range payments {
		err, txHash := Transfer(app.client, app.opts, app.senderPrivKey, payments[i].Address, payments[i].Amount)
		if err != nil {
			return payments, err
		}		
		payments[i].Transaction = txHash
		i++
	}
	return payments, nil
}
