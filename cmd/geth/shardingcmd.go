package main

import (
	"github.com/ethereum/go-ethereum/sharding"

	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingClientCommand = cli.Command{
		Action:      utils.MigrateFlags(shardingClient),
		Name:        "sharding",
		Aliases:     []string{"shard"},
		Usage:       "Start a sharding client",
		ArgsUsage:   "[endpoint]",
		Flags:       []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag},
		Category:    "SHARDING COMMANDS",
		Description: "TODO(prestonvanloon)- Add sharding client description",
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
