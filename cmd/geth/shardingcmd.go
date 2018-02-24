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
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding client that connects to a running geth node and proposes collations to a Validator Manager Contract. This feature is a work in progress.
`,
		Subcommands: []cli.Command{
			{
			Name: "joinvalidatorset",
			Usage: "Join validator set",
			ArgsUsage: "",
			Action: utils.MigrateFlags(joinValidatorSet),
			Category: "SHARDING COMMANDS",
			Description: `
Participate in validator set, client will deposit 100ETH from user's account into VMC validator set, which can be withdrawn at any time. This feature is a work in progress.
`,
		},
	},
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

func joinValidatorSet(ctx *cli.Context) error {
	return nil
}
