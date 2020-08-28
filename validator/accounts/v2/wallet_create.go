package v2

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
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
func CreateWalletCLI(cliCtx *cli.Context) (*Wallet, error) {
	keymanagerKind, err := inputKeymanagerKind(cliCtx)
	if err != nil {
		return nil, err
	}
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return nil, err
	}
	w, err := CreateWallet(cliCtx.Context, &WalletConfig{
		WalletDir:      walletDir,
		KeymanagerKind: keymanagerKind,
	})
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
	defaultOpts := direct.DefaultKeymanagerOpts()
	keymanagerConfig, err := direct.MarshalOptionsFile(context.Background(), defaultOpts)
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
	keymanagerConfig, err := derived.MarshalOptionsFile(ctx, derived.DefaultKeymanagerOpts())
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
	_, err = wallet.InitializeKeymanager(cliCtx.Context, skipMnemonic)
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

func inputKeymanagerKind(cliCtx *cli.Context) (v2keymanager.Kind, error) {
	if cliCtx.IsSet(flags.KeymanagerKindFlag.Name) {
		return v2keymanager.ParseKind(cliCtx.String(flags.KeymanagerKindFlag.Name))
	}
	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			keymanagerKindSelections[v2keymanager.Derived],
			keymanagerKindSelections[v2keymanager.Direct],
			keymanagerKindSelections[v2keymanager.Remote],
		},
	}
	selection, _, err := promptSelect.Run()
	if err != nil {
		return v2keymanager.Direct, fmt.Errorf("could not select wallet type: %v", formatPromptError(err))
	}
	return v2keymanager.Kind(selection), nil
}
