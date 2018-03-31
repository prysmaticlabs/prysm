package main

import (
	"github.com/ethereum/go-ethereum/sharding/collator"
	//"github.com/ethereum/go-ethereum/sharding/proposer"

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
	c := collator.NewCollatorClient(ctx)
	if err := collator.CollatorStart(c); err != nil {
		return err
	}
	c.Wait()
	return nil
}

func proposerClient(ctx *cli.Context) error {
	/*p := proposer.NewProposerClient(ctx)
	if err := proposer.ProposerStart(p); err != nil {
		return err
	}
	p.Wait()
	*/
	return nil
}
