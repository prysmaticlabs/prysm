package main

import (
	"os"
	"runtime"

	"github.com/prysmaticlabs/geth-sharding/beacon-chain/node"
	"github.com/prysmaticlabs/geth-sharding/beacon-chain/types"
	log "github.com/sirupsen/logrus"
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
	app.Flags = []cli.Flag{types.Web3ProviderFlag}

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
