package main

import (
	"github.com/ethereum/go-ethereum/sharding/notary"
	"github.com/ethereum/go-ethereum/sharding/proposer"

	"github.com/ethereum/go-ethereum/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	notaryClientCommand = cli.Command{
		Action:    utils.MigrateFlags(notaryClient),
		Name:      "sharding-notary",
		Aliases:   []string{"shard-notary"},
		Usage:     "Start a sharding notary client",
		ArgsUsage: "[endpoint]",
		Flags:     []cli.Flag{utils.DataDirFlag, utils.PasswordFileFlag, utils.NetworkIdFlag, utils.IPCPathFlag, utils.DepositFlag},
		Category:  "SHARDING COMMANDS",
		Description: `
Launches a sharding notary client that connects to a running geth node and submit collations to a Sharding Manager Contract. This feature is a work in progress.
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
Launches a sharding proposer client that connects to a running geth node and proposes collations to notary node. This feature is a work in progress.
`,
	}
)

func notaryClient(ctx *cli.Context) error {
	c := notary.NewNotary(ctx)
	return c.Start()
}

func proposerClient(ctx *cli.Context) error {
	p := proposer.NewProposer(ctx)
	return p.Start()
}
