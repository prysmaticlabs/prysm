package v2

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// EditWalletConfiguration for a user's on-disk wallet, being able to change
// things such as remote gRPC credentials for remote signing, derivation paths
// for HD wallets, and more.
func EditWalletConfiguration(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		return errors.New("no wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		return errors.Wrap(err, "could not parse wallet directory")
	}
	// Determine the keymanager kind for the wallet.
	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	if err != nil {
		return errors.Wrap(err, "could not select keymanager kind")
	}
	ctx := context.Background()
	wallet, err := OpenWallet(ctx, &WalletConfig{
		CanUnlockAccounts: false,
		WalletDir:         walletDir,
		KeymanagerKind:    keymanagerKind,
	})
	if err != nil {
		return errors.Wrap(err, "could not open wallet")
	}
	switch keymanagerKind {
	case v2keymanager.Direct:
		return errors.New("No configuration options available to edit for direct keymanager")
	case v2keymanager.Derived:
		return errors.New("Derived keymanager is not yet supported")
	case v2keymanager.Remote:
		enc, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			return errors.Wrap(err, "could not read config")
		}
		cfg, err := remote.UnmarshalConfigFile(enc)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal config")
		}
		log.Infof("Current configuration")
		fmt.Printf("%s\n", cfg)
		newCfg, err := inputRemoteKeymanagerConfig(cliCtx)
		if err != nil {
			return errors.Wrap(err, "could not get keymanager config")
		}
		encodedCfg, err := remote.MarshalConfigFile(ctx, newCfg)
		if err != nil {
			return errors.Wrap(err, "could not marshal config file")
		}
		if err := wallet.WriteKeymanagerConfigToDisk(ctx, encodedCfg); err != nil {
			return errors.Wrap(err, "could not write config to disk")
		}
	default:
		return fmt.Errorf("keymanager type %s is not supported", keymanagerKind)
	}
	return nil
}
