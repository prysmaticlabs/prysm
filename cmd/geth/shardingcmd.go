package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	node "github.com/ethereum/go-ethereum/sharding/node"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingCommand = cli.Command{
		Action:    utils.MigrateFlags(shardingCmd),
		Name:      "sharding",
		Usage:     "Start a sharding-enabled node",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.ActorFlag, utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding node that manages services related to submitting collations to a Sharding Manager Contract, notary and proposer services, and shardp2p connections. This feature is a work in progress.
`,
	}
)

// shardingCmd is the main cmd line entry point for starting a sharding-enabled node.
// A sharding node launches a suite of services including notary services,
// proposer services, and a shardp2p protocol.
func shardingCmd(ctx *cli.Context) error {
	// configures a sharding-enabled node using the cli's context.
	shardingNode, err := node.New(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize sharding node instance: %v", err)
	}
	defer shardingNode.Close()
	// starts a connection to a geth node and kicks off every registered service.
	if err := shardingNode.Start(); err != nil {
		return fmt.Errorf("Could not start sharding node: %v", err)
	}
	return nil
}
