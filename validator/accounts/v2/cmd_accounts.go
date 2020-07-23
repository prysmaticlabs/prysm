package v2

import (
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// AccountCommands for accounts-v2 for Prysm validators.
var AccountCommands = &cli.Command{
	Name:     "accounts-v2",
	Category: "accounts",
	Usage:    "defines commands for interacting with eth2 validator accounts (work in progress)",
	Subcommands: []*cli.Command{
		// AccountCommands for accounts-v2 for Prysm validators.
		{
			Name: "new",
			Description: `creates a new validator account for eth2. If no account exists at the wallet path, creates a new wallet for a user based on
specified input, capable of creating a direct, derived, or remote wallet.
this command outputs a deposit data string which is required to become a validator in eth2.`,
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
				flags.PasswordFileFlag,
				flags.SkipMnemonicConfirmFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				if err := NewAccount(cliCtx); err != nil {
					log.Fatalf("Could not create new account: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "list",
			Description: "Lists all validator accounts in a user's wallet directory",
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
				flags.ShowDepositDataFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				if err := ListAccounts(cliCtx); err != nil {
					log.Fatalf("Could not list accounts: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "export",
			Description: `exports the account of a given directory into a zip of the provided output path. This zip can be used to later import the account to another directory`,
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
				flags.BackupPathFlag,
				flags.AccountsFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				if err := ExportAccount(cliCtx); err != nil {
					log.Fatalf("Could not export accounts: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "import",
			Description: `imports the accounts from a given zip file to the provided wallet path. This zip can be created using the export command`,
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordsDirFlag,
				flags.BackupPathFlag,
				flags.PasswordFileFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				if err := ImportAccount(cliCtx); err != nil {
					log.Fatalf("Could not import accounts: %v", err)
				}
				return nil
			},
		},
	},
}
