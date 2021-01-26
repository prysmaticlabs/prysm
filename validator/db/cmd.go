package db

import (
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/urfave/cli/v2"
)

// DatabaseCommands for Prysm validator.
var DatabaseCommands = &cli.Command{
	Name:     "db",
	Category: "db",
	Usage:    "defines commands for interacting with eth2 validator database",
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
		{
			Name:     "migrate",
			Category: "db",
			Usage:    "Defines commands for running validator database migrations",
			Subcommands: []*cli.Command{
				{
					Name:  "up",
					Usage: "Runs up migrations for the validator database",
					Flags: cmd.WrapFlags([]cli.Flag{
						cmd.DataDirFlag,
					}),
					Before: tos.VerifyTosAcceptedOrPrompt,
					Action: func(cliCtx *cli.Context) error {
						if err := migrateUp(cliCtx); err != nil {
							log.Fatalf("Could not run database migrations: %v", err)
						}
						return nil
					},
				},
				{
					Name:  "down",
					Usage: "Runs down migrations for the validator database",
					Flags: cmd.WrapFlags([]cli.Flag{
						cmd.DataDirFlag,
					}),
					Before: tos.VerifyTosAcceptedOrPrompt,
					Action: func(cliCtx *cli.Context) error {
						if err := migrateDown(cliCtx); err != nil {
							log.Fatalf("Could not run database migrations: %v", err)
						}
						return nil
					},
				},
			},
		},
	},
}
