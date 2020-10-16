package accounts

import (
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// WalletCommands for accounts for Prysm validators.
var WalletCommands = &cli.Command{
	Name:     "wallet",
	Category: "wallet",
	Usage:    "defines commands for interacting with eth2 validator wallets (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Usage: "creates a new wallet with a desired type of keymanager: " +
				"either on-disk (imported), derived, or using remote credentials",
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.KeymanagerKindFlag,
				flags.GrpcRemoteAddressFlag,
				flags.RemoteSignerCertPathFlag,
				flags.RemoteSignerKeyPathFlag,
				flags.RemoteSignerCACertPathFlag,
				flags.WalletPasswordFileFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if _, err := CreateAndSaveWalletCli(cliCtx); err != nil {
					log.Fatalf("Could not create a wallet: %v", err)
				}
				return nil
			},
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := EditWalletConfigurationCli(cliCtx); err != nil {
					log.Fatalf("Could not edit wallet configuration: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "recover",
			Usage: "uses a derived wallet seed recovery phase to recreate an existing HD wallet",
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.MnemonicFileFlag,
				flags.WalletPasswordFileFlag,
				flags.NumAccountsFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := RecoverWalletCli(cliCtx); err != nil {
					log.Fatalf("Could not recover wallet: %v", err)
				}
				return nil
			},
		},
	},
}
