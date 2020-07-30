package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/urfave/cli/v2"
)

const phraseWordCount = 24

// RecoverWallet uses a menmonic seed phrase to recover a wallet into the path provided.
func RecoverWallet(cliCtx *cli.Context) error {
	mnemonic, err := inputMnemonic(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get mnemonic phrase")
	}
	wallet, err := NewWallet(cliCtx, v2keymanager.Derived)
	if err != nil {
		return errors.Wrap(err, "could not create new wallet")
	}
	ctx := context.Background()
	seedConfig, err := derived.SeedFileFromMnemonic(ctx, mnemonic, wallet.walletPassword)
	if err != nil {
		return errors.Wrap(err, "could not initialize new wallet seed file")
	}
	seedConfigFile, err := derived.MarshalEncryptedSeedFile(ctx, seedConfig)
	if err != nil {
		return errors.Wrap(err, "could not marshal encrypted wallet seed file")
	}
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
	if err := wallet.WriteEncryptedSeedToDisk(ctx, seedConfigFile); err != nil {
		return errors.Wrap(err, "could not write encrypted wallet seed config to disk")
	}
	keymanager, err := wallet.InitializeKeymanager(ctx, true)
	if err != nil {
		return err
	}
	km, ok := keymanager.(*derived.Keymanager)
	if !ok {
		return errors.New("not a derived keymanager")
	}

	numAccounts, err := inputNumAccounts(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get number of accounts to recover")
	}
	if numAccounts == 1 {
		if _, err := km.CreateAccount(ctx, true /*logAccountInfo*/); err != nil {
			return errors.Wrap(err, "could not create account in wallet")
		}
	} else {
		for i := 0; i < int(numAccounts); i++ {
			if _, err := km.CreateAccount(ctx, false /*logAccountInfo*/); err != nil {
				return errors.Wrap(err, "could not create account in wallet")
			}
		}
		log.WithField("wallet-path", wallet.AccountsDir()).Infof(
			"Successfully recovered HD wallet with %d accounts. Please use accounts-v2 list to view details for your accounts",
			numAccounts,
		)
		return nil
	}

	log.Infof(
		"Successfully recovered HD wallet and saved configuration to disk. " +
			"Make a new validator account with ./prysm.sh validator accounts-v2 create",
	)
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
	mnemonicPhrase, err := promptutil.ValidatePrompt("Enter the seed phrase for the wallet you would like to recover", validateMnemonic)
	if err != nil {
		return "", fmt.Errorf("could not get mnemonic phrase: %v", err)
	}
	return mnemonicPhrase, nil
}

func inputNumAccounts(cliCtx *cli.Context) (int64, error) {
	if cliCtx.IsSet(flags.NumAccountsFlag.Name) {
		numAccounts := cliCtx.Int64(flags.NumAccountsFlag.Name)
		return numAccounts, nil
	}
	numAccounts, err := promptutil.DefaultAndValidatePrompt("Enter how many accounts you would like to recover", "0", promptutil.ValidateNumber)
	if err != nil {
		return 0, err
	}
	numAccountsInt, err := strconv.Atoi(numAccounts)
	if err != nil {
		return 0, err
	}
	return int64(numAccountsInt), nil
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
		return fmt.Errorf("phrase must be %d words, entered %d", phraseWordCount, len(words))
	}
	return nil
}
