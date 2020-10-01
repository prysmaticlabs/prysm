package v2

import (
	"os"

	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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
			Name: "create",
			Description: `creates a new validator account for eth2. If no wallet exists at the given wallet path, creates a new wallet for a user based on
specified input, capable of creating a direct, derived, or remote wallet.
this command outputs a deposit data string which is required to become a validator in eth2.`,
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.AccountPasswordFileFlag,
				flags.NumAccountsFlag,
				flags.DeprecatedPasswordsDirFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
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
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.AccountPasswordFileFlag,
				flags.DeletePublicKeysFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
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
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.ShowDepositDataFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
				flags.DeprecatedPasswordsDirFlag,
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
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.WalletPasswordFileFlag,
				flags.BackupDirFlag,
				flags.BackupPublicKeysFlag,
				flags.BackupPasswordFile,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
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
			Flags: []cli.Flag{
				flags.WalletDirFlag,
				flags.KeysDirFlag,
				flags.WalletPasswordFileFlag,
				flags.AccountPasswordFileFlag,
				flags.ImportPrivateKeyFileFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
				flags.DeprecatedPasswordsDirFlag,
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
			Flags: []cli.Flag{
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
				featureconfig.SpadinaTestnet,
				featureconfig.ZinkenTestnet,
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
