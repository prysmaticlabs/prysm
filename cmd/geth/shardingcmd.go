package main

import (
	"github.com/ethereum/go-ethereum/sharding"

	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	collatorClientCommand = cli.Command{
		Action:    utils.MigrateFlags(collatorClient),
		Name:      "sharding-collator",
		Aliases:   []string{"shard-collator"},
		Usage:     "Start a sharding collator client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding collator client that connects to a running geth node and submit collations to a Sharding Manager Contract. This feature is a work in progress.
`,
	}
	proposerClientCommand = cli.Command{
		Action:    utils.MigrateFlags(proposerClient),
		Name:      "sharding-proposer",
		Aliases:   []string{"shard-proposer"},
		Usage:     "Start a sharding proposer client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding proposer client that connects to a running geth node and proposes collations to collator node. This feature is a work in progress.
`,
	}
)

func collatorClient(ctx *cli.Context) error {
	c := sharding.MakeCollatorClient(ctx)
	if err := c.Start(); err != nil {
		return err
	}
	c.Wait()
	return nil
}

func proposerClient(ctx *cli.Context) error {
	c := sharding.MakeProposerClient(ctx)
	if err := c.Start(); err != nil {
		return err
	}
	c.Wait()
	return nil

}
