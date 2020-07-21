package v2

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

const (
	minPasswordLength = 8
	// Min password score of 3 out of 5 based on the https://github.com/nbutton23/zxcvbn-go
	// library for strong-entropy password computation.
	minPasswordScore = 3
)

var keymanagerKindSelections = map[v2keymanager.Kind]string{
	v2keymanager.Derived: "HD Wallet (Recommended)",
	v2keymanager.Direct:  "Non-HD Wallet (Most Basic)",
	v2keymanager.Remote:  "Remote Signing Wallet (Advanced)",
}

// NewAccount creates a new validator account from user input by opening
// a wallet from the user's specified path.
func NewAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		log.Fatal("No wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(keymanagerKind)
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	switch keymanagerKind {
	case v2keymanager.Remote:
		log.Fatal("Cannot create a new account for a remote keymanager")
	case v2keymanager.Direct:
		passwordsDirPath := inputPasswordsDirectory(cliCtx)
		// Read the directory for password storage from user input.
		wallet, err := OpenWallet(ctx, &WalletConfig{
			PasswordsDir:      passwordsDirPath,
			WalletDir:         walletDir,
			CanUnlockAccounts: true,
		})
		if err != nil {
			log.Fatalf("Could not open wallet: %v", err)
		}
		configFile, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := direct.UnmarshalConfigFile(configFile)
		if err != nil {
			log.Fatal(err)
		}
		keymanager, err := direct.NewKeymanager(ctx, wallet, cfg, skipMnemonicConfirm)
		if err != nil {
			log.Fatal(err)
		}
		password, err := inputNewAccountPassword(cliCtx)
		if err != nil {
			log.Fatal(err)
		}
		// Create a new validator account using the specified keymanager.
		if _, err := keymanager.CreateAccount(ctx, password); err != nil {
			log.Fatalf("Could not create account in wallet: %v", err)
		}
	case v2keymanager.Derived:
		// Read the directory for password storage from user input.
		wallet, err := OpenWallet(ctx, &WalletConfig{
			WalletDir:         walletDir,
			CanUnlockAccounts: true,
		})
		if err != nil {
			log.Fatalf("Could not open wallet: %v", err)
		}
		configFile, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := derived.UnmarshalConfigFile(configFile)
		if err != nil {
			log.Fatal(err)
		}
		walletPassword, err := inputExistingWalletPassword()
		if err != nil {
			log.Fatal(err)
		}
		keymanager, err := derived.NewKeymanager(ctx, wallet, cfg, skipMnemonicConfirm, walletPassword)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := keymanager.CreateAccount(ctx); err != nil {
			log.Fatalf("Could not create account in wallet: %v", err)
		}
	default:
		log.Fatal("Keymanager kind not supported")
	}
	return nil
}
