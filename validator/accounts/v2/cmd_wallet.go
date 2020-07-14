package v2

import (
	"github.com/urfave/cli/v2"
)

// Commands for accounts-v2 for Prysm validators.
var Commands = &cli.Command{
	Name:     "wallet-v2",
	Category: "wallet-v2",
	Usage:    "defines commands for interacting with eth2 validator wallets (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name:        "accounts",
			Category:    "accounts",
			Usage:       "defines commands for interacting with eth2 validator accounts (work in progress)",
			Subcommands: AccountCommands,
		},
		{
			Name: "create",
			Usage: "creates a new wallet with a desired type of keymanager: " +
				"either on-disk (direct), derived, or using remote credentials",
			Action: CreateWallet,
		},
	},
}
