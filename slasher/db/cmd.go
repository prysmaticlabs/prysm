package db

import (
	"github.com/prysmaticlabs/prysm/runtime/tos"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/urfave/cli/v2"
)

// DatabaseCommands for Prysm slasher.
var DatabaseCommands = &cli.Command{
	Name:     "db",
	Category: "db",
	Usage:    "defines commands for interacting with the Prysm slasher database",
	Subcommands: []*cli.Command{
		{
			Name:        "restore",
			Description: `restores a database from a backup file`,
			Flags: cmd.WrapFlags([]cli.Flag{
				cmd.RestoreSourceFileFlag,
				cmd.RestoreTargetDirFlag,
			}),
			Before: tos.VerifyTosAcceptedOrPrompt,
			Action: func(cliCtx *cli.Context) error {
				if err := restore(cliCtx); err != nil {
					log.Fatalf("Could not restore database: %v", err)
				}
				return nil
			},
		},
	},
}
