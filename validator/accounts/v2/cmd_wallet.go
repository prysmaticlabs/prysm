package v2

import (
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/validator/flags"
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
			Action: CreateWallet,
		},
		{
			Name:  "edit-config",
			Usage: "edits a wallet configuration options, such as gRPC connection credentials and TLS certificates",
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.GrpcRemoteAddressFlag,
				flags.RemoteSignerCertPathFlag,
				flags.RemoteSignerKeyPathFlag,
				flags.RemoteSignerCACertPathFlag,
			},
			Action: EditWalletConfiguration,
		},
	},
}
