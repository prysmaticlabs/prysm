package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

const phraseWordCount = 24

func RecoverWallet(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil && !errors.Is(err, ErrNoWalletFound) {
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
				"edit your wallet configuration by running ./prysm.sh validator wallet-v2 edit",
		)
	}
	if err = recoverDerivedWallet(cliCtx, walletDir); err != nil {
		log.Fatalf("Could not initialize wallet with derived keymanager: %v", err)
	}
	log.WithField("wallet-path", walletDir).Infof(
		"Successfully recovered HD wallet and saved configuration to disk. " +
			"Make a new validator account with ./prysm.sh validator accounts-2 new",
	)
	return nil
}

func recoverDerivedWallet(cliCtx *cli.Context, walletDir string) error {
	mnemonic, err := inputMnemonic(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get mnemonic phrase")
	}
	passwordsDirPath := inputPasswordsDirectory(cliCtx)
	walletConfig := &WalletConfig{
		PasswordsDir:      passwordsDirPath,
		WalletDir:         walletDir,
		KeymanagerKind:    v2keymanager.Derived,
		CanUnlockAccounts: true,
	}
	ctx := context.Background()
	walletPassword, err := inputNewWalletPassword(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not input new wallet password")
	}

	seedConfig, err := derived.SeedFileFromMnemonic(ctx, mnemonic, walletPassword)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	if err != nil {
		return errors.Wrap(err, "could not marshal encrypted wallet seed file")
	}
	wallet, err := NewWallet(ctx, walletConfig)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}

	keymanagerConfig, err := derived.MarshalConfigFile(ctx, derived.DefaultConfig())
	if err != nil {
		return errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return errors.Wrap(err, "could not write keymanager config to disk")
	}
	if err := wallet.WriteEncryptedSeedToDisk(ctx, seedConfigFile); err != nil {
		return errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	return nil
}

func inputMnemonic(cliCtx *cli.Context) (string, error) {
	if cliCtx.IsSet(flags.MnemonicFileFlag.Name) {
		mnemonicFilePath := cliCtx.String(flags.MnemonicFileFlag.Name)
		data, err := ioutil.ReadFile(mnemonicFilePath)
		if err != nil {
			return "", err
		}
		enteredMnemonic := string(data)
		if err := validateMnemonic(enteredMnemonic); err != nil {
			return "", errors.Wrap(err, "mnemonic phrase did not pass validation")
		}
		return enteredMnemonic, nil
	}
	prompt := promptui.Prompt{
		Label:    "Enter the wallet recovery seed phrase you would like to recover",
		Validate: validateMnemonic,
	}
	menmonicPhrase, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine wallet directory: %v", formatPromptError(err))
	}
	return menmonicPhrase, nil
}

func validateMnemonic(mnemonic string) error {
	if strings.Trim(mnemonic, " ") == "" {
		return errors.New("phrase cannot be empty")
	}
	words := strings.Split(mnemonic, " ")
	for i, word := range words {
		if strings.Trim(word, " ") == "" {
			words = append(words[:i], words[i+1:]...)
		}
	}
	if len(words) != phraseWordCount {
		return fmt.Errorf("phrase must be 24 words, entered %d", len(words))
	}
	return nil
}
