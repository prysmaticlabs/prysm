package main

import (
	"github.com/ethereum/go-ethereum/sharding"

	"fmt"
	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	validatorClientCommand = cli.Command{
		Action:    utils.MigrateFlags(validatorClient),
		Name:      "sharding-validator",
		Aliases:   []string{"shard-validator"},
		Usage:     "Start a sharding validator client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding validator client that connects to a running geth node and proposes collations to a Validator Manager Contract. This feature is a work in progress.
`,
	}
	collatorClientCommand = cli.Command{
		Action:    utils.MigrateFlags(collatorClient),
		Name:      "sharding-collator",
		Aliases:   []string{"shard-collator"},
		Usage:     "Start a sharding collator client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding collator client that connects to a running geth node and proposes collations to validator node. This feature is a work in progress.
`,
	}
)

func validatorClient(ctx *cli.Context) error {
	c := sharding.MakeShardingClient(ctx)
	if err := c.Start(); err != nil {
		return err
	}
	c.Wait()
	return nil
}

func collatorClient(ctx *cli.Context) error {
	fmt.Println("Starting collator client")
	return nil
}
