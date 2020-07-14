package v2

import (
	"github.com/urfave/cli/v2"
)

// WalletCommands for accounts-v2 for Prysm validators.
var WalletCommands = &cli.Command{
	Name:     "wallet-v2",
	Category: "wallet-v2",
	Usage:    "defines commands for interacting with eth2 validator wallets (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Usage: "creates a new wallet with a desired type of keymanager: " +
				"either on-disk (direct), derived, or using remote credentials",
			Action: func(cliCtx *cli.Context) error {
				// Read a wallet's directory from user input.
				walletDir, err := inputWalletDir(cliCtx)
				if err != nil {
					log.Fatalf("Could not parse wallet directory: %v", err)
				}
				// Check if the user has a wallet at the specified path.
				// If a user does not have a wallet, we instantiate one
				// based on specified options.
				walletExists, err := hasDir(walletDir)
				if err != nil {
					log.Fatal(err)
				}
				if walletExists {
					log.Fatal(
						"You already have a wallet at the specified path. You can " +
							"edit your wallet configuration by running ./prysm.sh validator wallet-v2 edit",
					)
				}
				if _, err := CreateWallet(cliCtx, walletDir); err != nil {
					log.Fatalf("Could not create wallet: %v", err)
				}
				return nil
			},
		},
	},
}
