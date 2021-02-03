package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	incentives "github.com/videocoin/worker-incentives/src"
)

func main() {
	if os.Args[1] == "job" {
		job()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		cancel()
	}()
	app, err := incentives.NewApp(incentives.FromEnv())
	if err != nil {
		fmt.Printf("failed to bootstrap application %v\n", err)
		os.Exit(1)
	}

	if err := incentives.Serve(ctx, app); err != nil {
		if !errors.Is(err, context.Canceled) {
			fmt.Printf("application crashed with %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("application was stopped")
}
