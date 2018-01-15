package main

import (
	"github.com/ethereum/go-ethereum/sharding"

	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	shardingClientCommand = cli.Command{
		Action:      utils.MigrateFlags(shardingClient),
		Name:        "shard",
		Usage:       "Start a sharding client",
		ArgsUsage:   "[endpoint]",
		Flags:       append(consoleFlags, utils.DataDirFlag),
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
