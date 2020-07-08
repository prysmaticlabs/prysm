package v2

import (
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// Commands for accounts-v2 for Prysm validators.
var Commands = &cli.Command{
	Name:     "accounts-v2",
	Category: "accounts-v2",
	Usage:    "defines commands for interacting with eth2 validator accounts (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name: "new",
			Description: `creates a new validator account for eth2. If no account exists at the wallet path, creates a new wallet for a user based on
specified input, capable of creating a direct, derived, or remote wallet.
this command outputs a deposit data string which is required to become a validator in eth2.`,
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
			},
			Action: NewAccount,
		},
		{
			Name:        "list",
			Description: "Lists all validator accounts in a user's wallet directory",
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
				flags.ShowDepositDataFlag,
			},
			Action: ListAccounts,
		},
	},
}
