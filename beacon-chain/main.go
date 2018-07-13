package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/prysmaticlabs/geth-sharding/beacon-chain/node"
	"github.com/urfave/cli"
)

func startNode(ctx *cli.Context) error {
	beacon, err := node.New(ctx)
	if err != nil {
		return err
	}
	// starts a connection to a beacon node and kicks off every registered service.
	beacon.Start()
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "beacon-chain"
	app.Usage = "this is a beacon chain implementation for Ethereum 2.0"
	app.Action = startNode

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
