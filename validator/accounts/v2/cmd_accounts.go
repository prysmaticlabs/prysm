package v2

import (
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				flags.DeprecatedPasswordsDirFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := CreateAccount(cliCtx); err != nil {
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
				flags.WalletPasswordFileFlag,
				flags.ShowDepositDataFlag,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				flags.DeprecatedPasswordsDirFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := ListAccounts(cliCtx); err != nil {
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
				flags.BackupDirFlag,
				flags.BackupForPublicKeysFlag,
				flags.BackupsPasswordFile,
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := BackupAccounts(cliCtx); err != nil {
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
				featureconfig.AltonaTestnet,
				featureconfig.OnyxTestnet,
				flags.DeprecatedPasswordsDirFlag,
			},
			Action: func(cliCtx *cli.Context) error {
				featureconfig.ConfigureValidator(cliCtx)
				if err := ImportAccounts(cliCtx); err != nil {
					log.Fatalf("Could not import accounts: %v", err)
				}
				return nil
			},
		},
	},
}
