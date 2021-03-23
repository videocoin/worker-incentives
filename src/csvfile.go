package app

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
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

func etherFloatToWei(eth *big.Float) *big.Int {
	truncInt, _ := eth.Int(nil)
	truncInt = new(big.Int).Mul(truncInt, big.NewInt(params.Ether))
	fracStr := strings.Split(fmt.Sprintf("%.18f", eth), ".")[1]
	fracStr += strings.Repeat("0", 18-len(fracStr))
	fracInt, _ := new(big.Int).SetString(fracStr, 10)
	wei := new(big.Int).Add(truncInt, fracInt)
	return wei
}

func ReadPaymentsFile(inputfile string) ([]Payment, error) {
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
		if record[0] == "worker_address" {
			continue
		}
		fmt.Printf("Addressn:%s Amount:%s\n", record[0], record[1])
		if !common.IsHexAddress(record[0]) {
			fmt.Printf("Addressn:%s not recognized!\n", record[0])
			continue
		}

		floatAmount, ok := new(big.Float).SetString(record[1])
		if ok {
			Amount := etherFloatToWei(floatAmount)
			Address := common.HexToAddress(record[0])
			payment := Payment{ID: i, Address: Address, Amount: Amount}
			payments = append(payments, payment)
		} else {
			fmt.Printf("Failed converson-2 %v", record[1])
		}
	}
	return payments, nil
}

func WritePaymentsReceipt(outputfile string, payments []Payment) error {
	outfile, err := os.Create(outputfile)

	for _, payment := range payments {
		_, err = fmt.Fprintf(outfile, "%s,%v,%s\n", payment.Address.Hex(), payment.Amount, payment.Transaction.Hex())
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}
