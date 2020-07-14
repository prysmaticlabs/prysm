package v2

import (
	"context"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"

	"github.com/urfave/cli/v2"
)

func CreateWallet(cliCtx *cli.Context) error {
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
				"edit your wallet configuration by running ./prysm validator wallet-v2 edit",
		)
	}
	// Determine the desired keymanager kind for the wallet from user input.
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		log.Fatalf("Could not select keymanager kind: %v", err)
	}
	switch keymanagerKind {
	case v2keymanager.Direct:
		if err := initializeDirectWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not initialize wallet with direct keymanager: %v", err)
		}
		log.Infof(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm validator wallet-v2 accounts new",
		)
	case v2keymanager.Derived:
		log.Fatal("Derived keymanager is not yet supported")
	case v2keymanager.Remote:
		if err := initializeRemoteSignerWallet(cliCtx); err != nil {
			log.Fatalf("Could not initialize wallet with remote keymanager: %v", err)
		}
		log.Infof(
			"Successfully created wallet with remote keymanager configuration",
		)
	default:
		log.Fatalf("Keymanager type %s is not supported", keymanagerKind.String())
	}
	return nil
}

func initializeDirectWallet(cliCtx *cli.Context, walletDir string) error {
	passwordsDirPath := inputPasswordsDirectory(cliCtx)
	walletConfig := &WalletConfig{
		PasswordsDir:      passwordsDirPath,
		WalletDir:         walletDir,
		KeymanagerKind:    v2keymanager.Direct,
		CanUnlockAccounts: true,
	}
	ctx := context.Background()
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}
	keymanager, err := direct.NewKeymanager(ctx, wallet, direct.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not create new direct keymanager")
	}
	keymanagerConfig, err := keymanager.MarshalConfigFile(ctx)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	return wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig)
}

func initializeRemoteSignerWallet(cliCtx *cli.Context) error {
	return nil
}
