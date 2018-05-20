package main

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/sharding/client"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingCommand = cli.Command{
		Action:    utils.MigrateFlags(shardingClient),
		Name:      "sharding",
		Usage:     "Start a sharding-enabled node",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding client that submits collations to a Sharding Manager Contract, handles notary and proposer services, and manages shardp2p connections. This feature is a work in progress.
`,
	}
)

func shardingClient(ctx *cli.Context) error {
	// configures a sharding-enabled node using the cli's context.
	shardingNode := client.NewClient(ctx)
	return shardingNode.Start()
}

// func notaryClient(ctx *cli.Context) error {
// 	c := notary.NewNotary(ctx)
// 	return c.Start()
// }

// func proposerClient(ctx *cli.Context) error {
// 	p := proposer.NewProposer(ctx)
// 	return p.Start()
// }
