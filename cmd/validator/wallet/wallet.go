package wallet

import (
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "wallet")

// Commands for wallets for Prysm validators.
var Commands = &cli.Command{
	Name:     "wallet",
	Category: "wallet",
	Usage:    "defines commands for interacting with eth2 validator wallets",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Usage: "creates a new wallet with a desired type of keymanager: " +
				"either on-disk (imported), derived, or using remote credentials",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.KeymanagerKindFlag,
				flags.GrpcRemoteAddressFlag,
				flags.DisableRemoteSignerTlsFlag,
				flags.RemoteSignerCertPathFlag,
				flags.RemoteSignerKeyPathFlag,
				flags.RemoteSignerCACertPathFlag,
				flags.WalletPasswordFileFlag,
				flags.Mnemonic25thWordFileFlag,
				flags.SkipMnemonic25thWordCheckFlag,
				featureconfig.Mainnet,
				featureconfig.PyrmontTestnet,
				featureconfig.ToledoTestnet,
				featureconfig.PraterTestnet,
				cmd.AcceptTosFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
					return err
				}
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if _, err := accounts.CreateAndSaveWalletCli(cliCtx); err != nil {
					log.Fatalf("Could not create a wallet: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "edit-config",
			Usage: "edits a wallet configuration options, such as gRPC connection credentials and TLS certificates",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.GrpcRemoteAddressFlag,
				flags.DisableRemoteSignerTlsFlag,
				flags.RemoteSignerCertPathFlag,
				flags.RemoteSignerKeyPathFlag,
				flags.RemoteSignerCACertPathFlag,
				featureconfig.Mainnet,
				featureconfig.PyrmontTestnet,
				featureconfig.ToledoTestnet,
				featureconfig.PraterTestnet,
				cmd.AcceptTosFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
					return err
				}
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := accounts.EditWalletConfigurationCli(cliCtx); err != nil {
					log.Fatalf("Could not edit wallet configuration: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "recover",
			Usage: "uses a derived wallet seed recovery phase to recreate an existing HD wallet",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.MnemonicFileFlag,
				flags.WalletPasswordFileFlag,
				flags.NumAccountsFlag,
				flags.Mnemonic25thWordFileFlag,
				flags.SkipMnemonic25thWordCheckFlag,
				featureconfig.Mainnet,
				featureconfig.PyrmontTestnet,
				featureconfig.ToledoTestnet,
				featureconfig.PraterTestnet,
				cmd.AcceptTosFlag,
			}),
			Before: func(cliCtx *cli.Context) error {
				if err := cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags); err != nil {
					return err
				}
				return tos.VerifyTosAcceptedOrPrompt(cliCtx)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := accounts.RecoverWalletCli(cliCtx); err != nil {
					log.Fatalf("Could not recover wallet: %v", err)
				}
				return nil
			},
		},
	},
}
