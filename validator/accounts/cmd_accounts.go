package accounts

import (
	"os"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

// AccountCommands for Prysm validators.
var AccountCommands = &cli.Command{
	Name:     "accounts",
	Category: "accounts",
	Usage:    "defines commands for interacting with eth2 validator accounts (work in progress)",
	Subcommands: []*cli.Command{
		{
			Name: "create",
			Description: `creates a new validator account for eth2. If no wallet exists at the given wallet path, creates a new wallet for a user based on
specified input, capable of creating a imported, derived, or remote wallet.
this command outputs a deposit data string which is required to become a validator in eth2.`,
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.NumAccountsFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := CreateAccountCli(cliCtx); err != nil {
					log.Fatalf("Could not create new account: %v", err)
				}
				return nil
			},
		},
		{
			Name:        "delete",
			Description: `deletes the selected accounts from a users wallet.`,
			Flags: cmd.WrapFlags([]cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.DeletePublicKeysFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := DeleteAccountCli(cliCtx); err != nil {
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := ListAccountsCli(cliCtx); err != nil {
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := BackupAccountsCli(cliCtx); err != nil {
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := ImportAccountsCli(cliCtx); err != nil {
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.MedallaTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
			}),
			Before: func(cliCtx *cli.Context) error {
				return cmd.LoadFlagsFromConfig(cliCtx, cliCtx.Command.Flags)
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := ExitAccountsCli(cliCtx, os.Stdin); err != nil {
					log.Fatalf("Could not perform voluntary exit: %v", err)
				}
				return nil
			},
		},
	},
}
