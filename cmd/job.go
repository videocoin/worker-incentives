package main

import (
	"context"
	"fmt"
	"os"

	incentives "github.com/videocoin/worker-incentives/src"
)

func job() {
	conf := incentives.FromEnv()
	app, err := incentives.NewApp(conf)
	if err != nil {
		fmt.Printf("failed to bootstrap application %v\n", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), conf.JobTimeout)
	defer cancel()
	if err := app.Execute(ctx, conf.InputFileName, conf.OutputFileName); err != nil {
		app.Log().Fatalf("failed to execute request %v\n", err)
	}
	app.Log().Infof("job request finished succesfully")
}
