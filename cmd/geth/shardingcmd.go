package main

import (
	"github.com/ethereum/go-ethereum/sharding"

	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingClientCommand = cli.Command{
		Action:    utils.MigrateFlags(shardingClient),
		Name:      "sharding",
		Aliases:   []string{"shard"},
		Usage:     "Start a sharding client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.JoinValidatorSetFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding client that connects to a running geth node and proposes collations to a Validator Manager Contract. This feature is a work in progress.
`,
	}
)

func shardingClient(ctx *cli.Context) error {
	c := sharding.MakeShardingClient(ctx)
	if err := c.Start(); err != nil {
		return err
	}
	c.Wait()
	return nil
}
