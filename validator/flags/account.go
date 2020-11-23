package flags

import (
	"path/filepath"

	"github.com/urfave/cli/v2"
)

var (
	// Wallet flags.

	// WalletDirFlag defines the path to a wallet directory for Prysm accounts.
	WalletDirFlag = &cli.StringFlag{
		Name:  "wallet-dir",
		Usage: "Path to a wallet directory on-disk for Prysm validator accounts",
		Value: filepath.Join(DefaultValidatorDir(), WalletDefaultDirName),
	}
	// WalletPasswordFileFlag is the path to a file containing your wallet password.
	WalletPasswordFileFlag = &cli.StringFlag{
		Name:  "wallet-password-file",
		Usage: "Path to a plain-text, .txt file containing your wallet password",
	}

	// Account flags.

	// NumAccountsFlag defines the amount of accounts to generate for derived wallets.
	NumAccountsFlag = &cli.IntFlag{
		Name:  "num-accounts",
		Usage: "Number of accounts to generate for derived wallets",
		Value: 1,
	}
	// AccountPasswordFileFlag is path to a file containing a password for a validator account.
	AccountPasswordFileFlag = &cli.StringFlag{
		Name:  "account-password-file",
		Usage: "Path to a plain-text, .txt file containing a password for a validator account",
	}
	// BackupPasswordFile for encrypting accounts a user wishes to back up.
	BackupPasswordFile = &cli.StringFlag{
		Name:  "backup-password-file",
		Usage: "Path to a plain-text, .txt file containing the desired password for your backed up accounts",
		Value: "",
	}
	// MnemonicFileFlag is used to enter a file to mnemonic phrase for new wallet creation, non-interactively.
	MnemonicFileFlag = &cli.StringFlag{
		Name:  "mnemonic-file",
		Usage: "File to retrieve mnemonic for non-interactively passing a mnemonic phrase into wallet recover.",
	}
	// Mnemonic25thWordFileFlag defines a path to a file containing a "25th" word mnemonic passphrase for advanced users.
	Mnemonic25thWordFileFlag = &cli.StringFlag{
		Name:  "mnemonic-25th-word-file",
		Usage: "(Advanced) Path to a plain-text, .txt file containing a 25th word passphrase for your mnemonic for HD wallets",
	}
	// SkipMnemonic25thWordCheckFlag allows for skipping a check for mnemonic 25th word passphrases for HD wallets.
	SkipMnemonic25thWordCheckFlag = &cli.StringFlag{
		Name:  "skip-mnemonic-25th-word-check",
		Usage: "Allows for skipping the check for a mnemonic 25th word passphrase for HD wallets",
	}
	// ImportPrivateKeyFileFlag allows for directly importing a private key hex string as an account.
	ImportPrivateKeyFileFlag = &cli.StringFlag{
		Name:  "import-private-key-file",
		Usage: "Path to a plain-text, .txt file containing a hex string representation of a private key to import",
	}
	// KeymanagerKindFlag defines the kind of keymanager desired by a user during wallet creation.
	KeymanagerKindFlag = &cli.StringFlag{
		Name:  "keymanager-kind",
		Usage: "Kind of keymanager, either imported, derived, or remote, specified during wallet creation",
		Value: "",
	}

	// Account management flags.

	// DeletePublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to delete from their wallet.
	DeletePublicKeysFlag = &cli.StringFlag{
		Name:  "delete-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to delete",
		Value: "",
	}
	// DisablePublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to disable for their wallet.
	DisablePublicKeysFlag = &cli.StringFlag{
		Name:  "disable-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to disable",
		Value: "",
	}
	// EnablePublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to enable for their wallet.
	EnablePublicKeysFlag = &cli.StringFlag{
		Name:  "enable-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to enable",
		Value: "",
	}
	// BackupPublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts which a user desires to backup from their wallet.
	BackupPublicKeysFlag = &cli.StringFlag{
		Name:  "backup-public-keys",
		Usage: "Comma-separated list of public key hex strings to specify which validator accounts to backup",
		Value: "",
	}
	// VoluntaryExitPublicKeysFlag defines a comma-separated list of hex string public keys
	// for accounts on which a user wants to perform a voluntary exit.
	VoluntaryExitPublicKeysFlag = &cli.StringFlag{
		Name: "public-keys",
		Usage: "Comma-separated list of public key hex strings to specify on which validator accounts to perform " +
			"a voluntary exit",
		Value: "",
	}

	// Directory flags.

	// KeysDirFlag defines the path for a directory where keystores to be imported at stored.
	KeysDirFlag = &cli.StringFlag{
		Name:  "keys-dir",
		Usage: "Path to a directory where keystores to be imported are stored",
	}
	// BackupDirFlag defines the path for the zip backup of the wallet will be created.
	BackupDirFlag = &cli.StringFlag{
		Name:  "backup-dir",
		Usage: "Path to a directory where accounts will be backed up into a zip file",
		Value: DefaultValidatorDir(),
	}

	// Cosmetic flags.

	// ShowDepositDataFlag for accounts.
	ShowDepositDataFlag = &cli.BoolFlag{
		Name:  "show-deposit-data",
		Usage: "Display raw eth1 tx deposit data for validator accounts",
		Value: false,
	}
	// ShowPrivateKeysFlag for accounts.
	ShowPrivateKeysFlag = &cli.BoolFlag{
		Name:  "show-private-keys",
		Usage: "Display the private keys for validator accounts",
		Value: false,
	}
	// SkipDepositConfirmationFlag skips the y/n confirmation prompt for sending a deposit to the deposit contract.
	SkipDepositConfirmationFlag = &cli.BoolFlag{
		Name:  "skip-deposit-confirmation",
		Usage: "Skips the y/n confirmation prompt for sending a deposit to the deposit contract",
		Value: false,
	}
)
