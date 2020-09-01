package v2

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/remote"
	"github.com/urfave/cli/v2"
)

// CreateWalletConfig defines the parameters needed to call the create wallet functions.
type CreateWalletConfig struct {
	WalletCfg            *WalletConfig
	RemoteKeymanagerOpts *remote.KeymanagerOpts
	SkipMnemonicConfirm  bool
}

// CreateAndSaveWalletCli from user input with a desired keymanager. If a
// wallet already exists in the path, it suggests the user alternatives
// such as how to edit their existing wallet configuration.
func CreateAndSaveWalletCli(cliCtx *cli.Context) (*Wallet, error) {
	keymanagerKind, err := extractKeymanagerKindFromCli(cliCtx)
	if err != nil {
		return nil, err
	}
	createWalletConfig, err := extractWalletCreationConfigFromCli(cliCtx, keymanagerKind)
	if err != nil {
		return nil, err
	}
	return CreateWalletWithKeymanager(cliCtx.Context, createWalletConfig)
}

// CreateWalletWithKeymanager specified by configuration options.
func CreateWalletWithKeymanager(ctx context.Context, cfg *CreateWalletConfig) (*Wallet, error) {
	if err := WalletExists(cfg.WalletCfg.WalletDir); err != nil {
		if !errors.Is(err, ErrNoWalletFound) {
			return nil, errors.Wrap(err, "could not check if wallet exists")
		}
	}
	accountsPath := filepath.Join(cfg.WalletCfg.WalletDir, cfg.WalletCfg.KeymanagerKind.String())
	w := &Wallet{
		accountsPath:   accountsPath,
		keymanagerKind: cfg.WalletCfg.KeymanagerKind,
		walletDir:      cfg.WalletCfg.WalletDir,
		walletPassword: cfg.WalletCfg.WalletPassword,
	}
	var err error
	switch w.KeymanagerKind() {
	case v2keymanager.Direct:
		if err = createDirectKeymanagerWallet(ctx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with direct keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created wallet with on-disk keymanager configuration. " +
				"Make a new validator account with ./prysm.sh validator accounts-v2 create",
		)
	case v2keymanager.Derived:
		if err = createDerivedKeymanagerWallet(ctx, w, cfg.SkipMnemonicConfirm); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet with derived keymanager")
		}
		log.WithField("--wallet-dir", w.walletDir).Info(
			"Successfully created HD wallet and saved configuration to disk. " +
				"Make a new validator account with ./prysm.sh validator accounts-2 create",
		)
	case v2keymanager.Remote:
		if err = createRemoteKeymanagerWallet(ctx, w, cfg.RemoteKeymanagerOpts); err != nil {
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

func extractKeymanagerKindFromCli(cliCtx *cli.Context) (v2keymanager.Kind, error) {
	return inputKeymanagerKind(cliCtx)
}

func extractWalletCreationConfigFromCli(cliCtx *cli.Context, keymanagerKind v2keymanager.Kind) (*CreateWalletConfig, error) {
	walletDir, err := inputDirectory(cliCtx, walletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return nil, err
	}
	walletPassword, err := inputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		newWalletPasswordPromptText,
		true, /* Should confirm password */
		promptutil.ValidatePasswordInput,
	)
	if err != nil {
		return nil, err
	}
	createWalletConfig := &CreateWalletConfig{
		WalletCfg: &WalletConfig{
			WalletDir:      walletDir,
			KeymanagerKind: keymanagerKind,
			WalletPassword: walletPassword,
		},
		SkipMnemonicConfirm: cliCtx.Bool(flags.SkipDepositConfirmationFlag.Name),
	}

	if keymanagerKind == v2keymanager.Remote {
		opts, err := inputRemoteKeymanagerConfig(cliCtx)
		if err != nil {
			return nil, errors.Wrap(err, "could not input remote keymanager config")
		}
		createWalletConfig.RemoteKeymanagerOpts = opts
	}
	return createWalletConfig, nil
}

func createDirectKeymanagerWallet(ctx context.Context, wallet *Wallet) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	defaultOpts := direct.DefaultKeymanagerOpts()
	keymanagerConfig, err := direct.MarshalOptionsFile(ctx, defaultOpts)
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	return nil
}

func createDerivedKeymanagerWallet(ctx context.Context, wallet *Wallet, skipMnemonicConfirm bool) error {
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
	_, err = wallet.InitializeKeymanager(ctx, skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not initialize keymanager")
	}
	return nil
}

func createRemoteKeymanagerWallet(ctx context.Context, wallet *Wallet, opts *remote.KeymanagerOpts) error {
	keymanagerConfig, err := remote.MarshalOptionsFile(ctx, opts)
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
