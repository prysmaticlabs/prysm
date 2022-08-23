package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/v3/io/prompt"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/iface"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/derived"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/local"
	"github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote"
	remoteweb3signer "github.com/prysmaticlabs/prysm/v3/validator/keymanager/remote-web3signer"
	"github.com/urfave/cli/v2"
)

const (
	// #nosec G101 -- Not sensitive data
	newMnemonicPassphraseYesNoText = "(Advanced) Do you want to setup a '25th word' passphrase for your mnemonic? [y/n]"
	// #nosec G101 -- Not sensitive data
	newMnemonicPassphrasePromptText = "(Advanced) Setup a passphrase '25th word' for your mnemonic " +
		"(WARNING: You cannot recover your keys from your mnemonic if you forget this passphrase!)"
)

// CreateWalletConfig defines the parameters needed to call the create wallet functions.
type CreateWalletConfig struct {
	SkipMnemonicConfirm   bool
	NumAccounts           int
	RemoteKeymanagerOpts  *remote.KeymanagerOpts
	Web3SignerSetupConfig *remoteweb3signer.SetupConfig
	WalletCfg             *wallet.Config
	Mnemonic25thWord      string
}

// CreateAndSaveWalletCli from user input with a desired keymanager. If a
// wallet already exists in the path, it suggests the user alternatives
// such as how to edit their existing wallet configuration.
func CreateAndSaveWalletCli(cliCtx *cli.Context) (*wallet.Wallet, error) {
	keymanagerKind, err := extractKeymanagerKindFromCli(cliCtx)
	if err != nil {
		return nil, err
	}
	createWalletConfig, err := ExtractWalletCreationConfigFromCli(cliCtx, keymanagerKind)
	if err != nil {
		return nil, err
	}

	dir := createWalletConfig.WalletCfg.WalletDir
	dirExists, err := wallet.Exists(dir)
	if err != nil {
		return nil, err
	}
	if dirExists {
		return nil, errors.New("a wallet already exists at this location. Please input an" +
			" alternative location for the new wallet or remove the current wallet")
	}

	w, err := CreateWalletWithKeymanager(cliCtx.Context, createWalletConfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not create wallet")
	}
	return w, nil
}

// CreateWalletWithKeymanager specified by configuration options.
func CreateWalletWithKeymanager(ctx context.Context, cfg *CreateWalletConfig) (*wallet.Wallet, error) {
	w := wallet.New(&wallet.Config{
		WalletDir:      cfg.WalletCfg.WalletDir,
		KeymanagerKind: cfg.WalletCfg.KeymanagerKind,
		WalletPassword: cfg.WalletCfg.WalletPassword,
	})
	var err error
	switch w.KeymanagerKind() {
	case keymanager.Local:
		if err = CreateLocalKeymanagerWallet(ctx, w); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		// TODO(#9883) - Remove this when we have a better way to handle this. should be safe to use for now.
		km, err := w.InitializeKeymanager(ctx, iface.InitKeymanagerConfig{ListenForChanges: false})
		if err != nil {
			return nil, errors.Wrap(err, ErrCouldNotInitializeKeymanager)
		}
		localKm, ok := km.(*local.Keymanager)
		if !ok {
			return nil, errors.Wrap(err, ErrCouldNotInitializeKeymanager)
		}
		accountsKeystore, err := localKm.CreateAccountsKeystore(ctx, make([][]byte, 0), make([][]byte, 0))
		if err != nil {
			return nil, err
		}
		encodedAccounts, err := json.MarshalIndent(accountsKeystore, "", "\t")
		if err != nil {
			return nil, err
		}
		if err = w.WriteFileAtPath(ctx, local.AccountsPath, local.AccountsKeystoreFileName, encodedAccounts); err != nil {
			return nil, err
		}

		log.WithField("--wallet-dir", cfg.WalletCfg.WalletDir).Info(
			"Successfully created wallet with ability to import keystores",
		)
	case keymanager.Derived:
		if err = createDerivedKeymanagerWallet(
			ctx,
			w,
			cfg.Mnemonic25thWord,
			cfg.SkipMnemonicConfirm,
			cfg.NumAccounts,
		); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		log.WithField("--wallet-dir", cfg.WalletCfg.WalletDir).Info(
			"Successfully created HD wallet from mnemonic and regenerated accounts",
		)
	case keymanager.Remote:
		if err = createRemoteKeymanagerWallet(ctx, w, cfg.RemoteKeymanagerOpts); err != nil {
			return nil, errors.Wrap(err, "could not initialize wallet")
		}
		log.WithField("--wallet-dir", cfg.WalletCfg.WalletDir).Info(
			"Successfully created wallet with remote keymanager configuration",
		)
	case keymanager.Web3Signer:
		return nil, errors.New("web3signer keymanager does not require persistent wallets.")
	default:
		return nil, errors.Wrapf(err, errKeymanagerNotSupported, w.KeymanagerKind())
	}
	return w, nil
}

func extractKeymanagerKindFromCli(cliCtx *cli.Context) (keymanager.Kind, error) {
	return inputKeymanagerKind(cliCtx)
}

// ExtractWalletCreationConfigFromCli prompts the user for wallet creation input.
func ExtractWalletCreationConfigFromCli(cliCtx *cli.Context, keymanagerKind keymanager.Kind) (*CreateWalletConfig, error) {
	walletDir, err := userprompt.InputDirectory(cliCtx, userprompt.WalletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return nil, err
	}
	walletPassword, err := prompt.InputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		wallet.NewWalletPasswordPromptText,
		wallet.ConfirmPasswordPromptText,
		true, /* Should confirm password */
		prompt.ValidatePasswordInput,
	)
	if err != nil {
		return nil, err
	}
	createWalletConfig := &CreateWalletConfig{
		WalletCfg: &wallet.Config{
			WalletDir:      walletDir,
			KeymanagerKind: keymanagerKind,
			WalletPassword: walletPassword,
		},
		SkipMnemonicConfirm: cliCtx.Bool(flags.SkipDepositConfirmationFlag.Name),
	}
	skipMnemonic25thWord := cliCtx.IsSet(flags.SkipMnemonic25thWordCheckFlag.Name)
	has25thWordFile := cliCtx.IsSet(flags.Mnemonic25thWordFileFlag.Name)
	if keymanagerKind == keymanager.Derived {
		numAccounts, err := inputNumAccounts(cliCtx)
		if err != nil {
			return nil, errors.Wrap(err, "could not get number of accounts to generate")
		}
		createWalletConfig.NumAccounts = int(numAccounts)
	}
	if keymanagerKind == keymanager.Derived && !skipMnemonic25thWord && !has25thWordFile {
		resp, err := prompt.ValidatePrompt(
			os.Stdin, newMnemonicPassphraseYesNoText, prompt.ValidateYesOrNo,
		)
		if err != nil {
			return nil, errors.Wrap(err, "could not validate choice")
		}
		if strings.EqualFold(resp, "y") {
			mnemonicPassphrase, err := prompt.InputPassword(
				cliCtx,
				flags.Mnemonic25thWordFileFlag,
				newMnemonicPassphrasePromptText,
				"Confirm mnemonic passphrase",
				true, /* Should confirm password */
				func(input string) error {
					if strings.TrimSpace(input) == "" {
						return errors.New("input cannot be empty")
					}
					return nil
				},
			)
			if err != nil {
				return nil, err
			}
			createWalletConfig.Mnemonic25thWord = mnemonicPassphrase
		}
	}
	if keymanagerKind == keymanager.Remote {
		opts, err := userprompt.InputRemoteKeymanagerConfig(cliCtx)
		if err != nil {
			return nil, errors.Wrap(err, "could not input remote keymanager config")
		}
		createWalletConfig.RemoteKeymanagerOpts = opts
	}
	if keymanagerKind == keymanager.Web3Signer {
		return nil, errors.New("web3signer keymanager does not require persistent wallets.")
	}
	return createWalletConfig, nil
}

func CreateLocalKeymanagerWallet(_ context.Context, wallet *wallet.Wallet) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	return nil
}

func createDerivedKeymanagerWallet(
	ctx context.Context,
	wallet *wallet.Wallet,
	mnemonicPassphrase string,
	skipMnemonicConfirm bool,
	numAccounts int,
) error {
	if wallet == nil {
		return errors.New("nil wallet")
	}
	if err := wallet.SaveWallet(); err != nil {
		return errors.Wrap(err, "could not save wallet to disk")
	}
	km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
		Wallet:           wallet,
		ListenForChanges: true,
	})
	if err != nil {
		return errors.Wrap(err, "could not initialize HD keymanager")
	}
	mnemonic, err := derived.GenerateAndConfirmMnemonic(skipMnemonicConfirm)
	if err != nil {
		return errors.Wrap(err, "could not confirm mnemonic")
	}
	if err := km.RecoverAccountsFromMnemonic(ctx, mnemonic, mnemonicPassphrase, numAccounts); err != nil {
		return errors.Wrap(err, "could not recover accounts from mnemonic")
	}
	return nil
}

func createRemoteKeymanagerWallet(ctx context.Context, wallet *wallet.Wallet, opts *remote.KeymanagerOpts) error {
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

func inputKeymanagerKind(cliCtx *cli.Context) (keymanager.Kind, error) {
	if cliCtx.IsSet(flags.KeymanagerKindFlag.Name) {
		return keymanager.ParseKind(cliCtx.String(flags.KeymanagerKindFlag.Name))
	}
	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			wallet.KeymanagerKindSelections[keymanager.Local],
			wallet.KeymanagerKindSelections[keymanager.Derived],
			wallet.KeymanagerKindSelections[keymanager.Remote],
			wallet.KeymanagerKindSelections[keymanager.Web3Signer],
		},
	}
	selection, _, err := promptSelect.Run()
	if err != nil {
		return keymanager.Local, fmt.Errorf("could not select wallet type: %w", userprompt.FormatPromptError(err))
	}
	return keymanager.Kind(selection), nil
}

// TODO(mikeneuder): Remove duplicate function when migration wallet create
// to cmd/validator/wallet.
func inputNumAccounts(cliCtx *cli.Context) (int64, error) {
	if cliCtx.IsSet(flags.NumAccountsFlag.Name) {
		numAccounts := cliCtx.Int64(flags.NumAccountsFlag.Name)
		if numAccounts <= 0 {
			return 0, errors.New("must recover at least 1 account")
		}
		return numAccounts, nil
	}
	numAccounts, err := prompt.ValidatePrompt(os.Stdin, "Enter how many accounts you would like to generate from the mnemonic", prompt.ValidateNumber)
	if err != nil {
		return 0, err
	}
	numAccountsInt, err := strconv.Atoi(numAccounts)
	if err != nil {
		return 0, err
	}
	if numAccountsInt <= 0 {
		return 0, errors.New("must recover at least 1 account")
	}
	return int64(numAccountsInt), nil
}
