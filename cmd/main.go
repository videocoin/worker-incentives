package main

import (
	//"context"
	//"errors"
	//"fmt"
	"os"
	//"os/signal"

	//incentives "github.com/videocoin/worker-incentives/src"
)

func main() {
	if os.Args[1] == "job" {
		job()
		return
	}
}
