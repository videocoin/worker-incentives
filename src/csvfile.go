package incentives

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"math/big"
	"math"	
	"github.com/ethereum/go-ethereum/common"
	"strconv"
)

const (
	Wei   = 1
	GWei  = 1e9
	Ether = 1e18
)

func etherToWei(val *big.Int) *big.Int {
	return new(big.Int).Mul(val, big.NewInt(Ether))
}

type Payment struct {
	ID          int            `json:"id"`
	Address     common.Address `json:"address"`
	Amount      *big.Int       `json:"amount"`
	Transaction common.Hash    `json:"tx"`
}

func (app *App)ReadPaymentsFile(ctx context.Context, inputfile string) ([] Payment, error) {
	payments := []Payment{}
	csvfile, err := os.Open(inputfile)
	if err != nil {
			log.Fatalln("Couldn't open the csv file", err)
	}

	r := csv.NewReader(csvfile)
	var i = 0
	for {
			record, err := r.Read()
			if err == io.EOF {
					break
			}
			if err != nil {
					log.Fatal(err)
			}
		fmt.Printf("Addressn:%s Amount:%s\n", record[0], record[1])
		if (!common.IsHexAddress(record[0])) {
			fmt.Printf("Addressn:%s not recognized!\n", record[0]);
			continue
		}
		s, err := strconv.ParseFloat(record[1], 32) 
		if err != nil{
			fmt.Println("Failed converson;", s) 
		}
		//Amount, _ := new(big.Int).SetString(record[1], 10)
		EthAmount := new(big.Int).SetInt64(int64(math.Trunc(s)))
		Amount := etherToWei(EthAmount)
		Address := common.HexToAddress(record[0])
		payment := Payment{ID: i, Address: Address, Amount: Amount }
		payments = append(payments, payment)
	}
	return payments, nil
}

func (app *App)WritePaymentsReceipt(ctx context.Context, outputfile string, payments [] Payment) error {
	return nil
}


