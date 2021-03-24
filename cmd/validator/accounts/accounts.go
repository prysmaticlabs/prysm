package accounts

import (
	"os"

	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/tos"
	"github.com/prysmaticlabs/prysm/validator/accounts"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts")

// Commands for managing Prysm validator accounts.
var Commands = &cli.Command{
	Name:     "accounts",
	Category: "accounts",
	Usage:    "defines commands for interacting with eth2 validator accounts",
	Subcommands: []*cli.Command{
		{
			Name:        "delete",
			Description: `deletes the selected accounts from a users wallet.`,
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.DeletePublicKeysFlag,
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
				if err := accounts.DeleteAccountCli(cliCtx); err != nil {
					log.Fatalf("Could not delete account: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "list",
			Description: "Lists all validator accounts in a user's wallet directory",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.ShowDepositDataFlag,
				flags.ShowPrivateKeysFlag,
				flags.ListValidatorIndices,
				flags.BeaconRPCProviderFlag,
				cmd.GrpcMaxCallRecvMsgSizeFlag,
				flags.CertFlag,
				flags.GrpcHeadersFlag,
				flags.GrpcRetriesFlag,
				flags.GrpcRetryDelayFlag,
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
				if err := accounts.ListAccountsCli(cliCtx); err != nil {
					log.Fatalf("Could not list accounts: %v", err)
				}
				return nil
			},
		},
		{
			Name: "backup",
			Description: "backup accounts into EIP-2335 compliant keystore.json files zipped into a backup.zip file " +
				"at a desired output directory. Accounts to backup can also " +
				"be specified programmatically via a --backup-for-public-keys flag which specifies a comma-separated " +
				"list of hex string public keys",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.BackupDirFlag,
				flags.BackupPublicKeysFlag,
				flags.BackupPasswordFile,
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
				if err := accounts.BackupAccountsCli(cliCtx); err != nil {
					log.Fatalf("Could not backup accounts: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "import",
			Description: `imports eth2 validator accounts stored in EIP-2335 keystore.json files from an external directory`,
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.KeysDirFlag,
				flags.WalletPasswordFileFlag,
				flags.AccountPasswordFileFlag,
				flags.ImportPrivateKeyFileFlag,
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
				if err := accounts.ImportAccountsCli(cliCtx); err != nil {
					log.Fatalf("Could not import accounts: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "voluntary-exit",
			Description: "Performs a voluntary exit on selected accounts",
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.AccountPasswordFileFlag,
				flags.VoluntaryExitPublicKeysFlag,
				flags.BeaconRPCProviderFlag,
				cmd.GrpcMaxCallRecvMsgSizeFlag,
				flags.CertFlag,
				flags.GrpcHeadersFlag,
				flags.GrpcRetriesFlag,
				flags.GrpcRetryDelayFlag,
				flags.ExitAllFlag,
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
				if err := accounts.ExitAccountsCli(cliCtx, os.Stdin); err != nil {
					log.Fatalf("Could not perform voluntary exit: %v", err)
				}
				return nil
			},
		},
	},
}
