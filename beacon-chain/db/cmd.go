package db

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// DatabaseCommands for Prysm beacon node.
var DatabaseCommands = &cli.Command{
	Name:     "db",
	Category: "db",
	Usage:    "defines commands for interacting with eth2 beacon node database",
	Subcommands: []*cli.Command{
		{
			Name:        "restore",
			Description: `restores a database from a backup file`,
			Flags: cmd.WrapFlags([]cli.Flag{
				cmd.RestoreFromFileFlag,
				cmd.DataDirFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				fromFilePath := cliCtx.String(cmd.RestoreFromFileFlag.Name)
				datadir := cliCtx.String(cmd.DataDirFlag.Name)
				if err := kv.Restore(cliCtx.Context, fromFilePath, datadir); err != nil {
					logrus.Fatalf("Could not restore database: %v", err)
				}
				logrus.Info("Database restored successfully")
				return nil
			},
		},
	},
}
