package v2

import (
	"context"
	"encoding/json"

	remotehttp "github.com/bloxapp/key-vault/keymanager"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// CreateWallet from user input with a desired keymanager. If a
// wallet already exists in the path, it suggests the user alternatives
// such as how to edit their existing wallet configuration.
func CreateWallet(cliCtx *cli.Context) (*Wallet, error) {
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		return nil, err
	}
	w, err := NewWallet(cliCtx, keymanagerKind)
	if err != nil && !errors.Is(err, ErrWalletExists) {
		return nil, errors.Wrap(err, "could not check if wallet directory exists")
	}
	if errors.Is(err, ErrWalletExists) {
		return nil, ErrWalletExists
	}
	switch w.KeymanagerKind() {
	case v2keymanager.Direct:
		if err = createDirectKeymanagerWallet(cliCtx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with direct keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm.sh validator accounts-v2 create",
		)
	case v2keymanager.Derived:
		if err = createDerivedKeymanagerWallet(cliCtx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with derived keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created HD wallet and saved configuration to disk. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 create",
		)
	case v2keymanager.Remote:
		if err = createRemoteKeymanagerWallet(cliCtx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with remote keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created wallet with remote keymanager configuration",
		)
	case v2keymanager.RemoteHTTP:
		if err = createRemoteHTTPKeymanagerWallet(cliCtx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with remote HTTP keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created wallet with remote HTTP keymanager configuration",
		)
	default:
		return nil, errors.Wrapf(err, "keymanager type %s is not supported", w.KeymanagerKind())
	}
	return w, nil
}

func createDirectKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	defaultConfig := direct.DefaultConfig()
	keymanagerConfig, err := direct.MarshalConfigFile(context.Background(), defaultConfig)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(context.Background(), keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func createDerivedKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	ctx := context.Background()
	keymanagerConfig, err := derived.MarshalConfigFile(ctx, derived.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	skipMnemonic := cliCtx.Bool(flags.SkipDepositConfirmationFlag.Name)
	_, err = wallet.InitializeKeymanager(cliCtx, skipMnemonic)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	return nil
}

func createRemoteKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	conf, err := inputRemoteKeymanagerConfig(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not input remote keymanager config")
	}
	ctx := context.Background()
	keymanagerConfig, err := remote.MarshalConfigFile(ctx, conf)
	if err != nil {
		return errors.Wrap(err, "could not marshal config file")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func createRemoteHTTPKeymanagerWallet(cliCtx *cli.Context, wallet *Wallet) error {
	conf, err := inputRemoteHTTPKeymanagerConfig(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not input remote HTTP keymanager config")
	}

	keymanagerConfig, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "could not marshal config file")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(context.Background(), keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}
