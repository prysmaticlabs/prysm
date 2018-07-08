package main

import (
	"fmt"
	"os"

	"github.com/prysmaticlabs/geth-sharding/sharding/node"
	"github.com/prysmaticlabs/geth-sharding/sharding/utils"
	"github.com/urfave/cli"
)

func startNode(ctx *cli.Context) error {
	shardingNode, err := node.New(ctx)
	if err != nil {
		return err
	}
	// starts a connection to a beacon node and kicks off every registered service.
	shardingNode.Start()
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "sharding"
	app.Description = `
Launches a sharding node that manages services related to submitting collations to a Sharding Manager Contract, notary and proposer services, and shardp2p connections. This feature is a work in progress.
`
	app.Action = startNode
	app.Flags = []cli.Flag{utils.ActorFlag, utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag, utils.ShardIDFlag}
	if err := app.Run(os.Args); err != nil {
		if _, err := fmt.Fprintln(os.Stderr, err); err != nil {
			panic(err)
		}
		os.Exit(1)
	}
}
