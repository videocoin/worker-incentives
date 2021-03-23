package main

import (
	"fmt"
	"os"

	app "github.com/videocoin/worker-incentives/src"
)

func main() {
	if err := app.RootCommand().Execute(); err != nil {
		fmt.Printf("worker-incentives execution failed with %v\n", err)
		os.Exit(1)
	}
}
