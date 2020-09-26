package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/v2/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
	"github.com/urfave/cli/v2"
)

const phraseWordCount = 24

// RecoverWalletConfig to run the recover wallet function.
type RecoverWalletConfig struct {
	WalletDir      string
	WalletPassword string
	Mnemonic       string
	NumAccounts    int64
}

// RecoverWalletCli uses a menmonic seed phrase to recover a wallet into the path provided. This
// uses the CLI to extract necessary values to run the function.
func RecoverWalletCli(cliCtx *cli.Context) error {
	mnemonic, err := inputMnemonic(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get mnemonic phrase")
	}
	walletDir, err := prompt.InputDirectory(cliCtx, prompt.WalletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return err
	}
	walletPassword, err := promptutil.InputPassword(
		cliCtx,
		flags.WalletPasswordFileFlag,
		wallet.NewWalletPasswordPromptText,
		wallet.ConfirmPasswordPromptText,
		true, /* Should confirm password */
		promptutil.ValidatePasswordInput,
	)
	if err != nil {
		return err
	}
	numAccounts, err := inputNumAccounts(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get number of accounts to recover")
	}
	w, err := RecoverWallet(cliCtx.Context, &RecoverWalletConfig{
		WalletDir:      walletDir,
		WalletPassword: walletPassword,
		Mnemonic:       mnemonic,
		NumAccounts:    numAccounts,
	})
	if err != nil {
		return err
	}
	// We store the hashed password to disk.
	if err := w.SaveHashedPassword(cliCtx.Context); err != nil {
		return err
	}
	log.Infof(
		"Successfully recovered HD wallet and saved configuration to disk. " +
			"Make a new validator account with ./prysm.sh validator accounts-v2 create",
	)
	return nil
}

// RecoverWallet uses a menmonic seed phrase to recover a wallet into the path provided.
func RecoverWallet(ctx context.Context, cfg *RecoverWalletConfig) (*wallet.Wallet, error) {
	// Ensure that the wallet directory does not contain a wallet already
	dirExists, err := wallet.Exists(cfg.WalletDir)
	if err != nil {
		return nil, err
	}
	if dirExists {
		return nil, errors.New("a wallet already exists at this location. Please input an" +
			" alternative location for the new wallet or remove the current wallet")
	}
	w := wallet.New(&wallet.Config{
		WalletDir:      cfg.WalletDir,
		KeymanagerKind: v2keymanager.Derived,
		WalletPassword: cfg.WalletPassword,
	})
	keymanagerConfig, err := derived.MarshalOptionsFile(ctx, derived.DefaultKeymanagerOpts())
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := w.SaveWallet(); err != nil {
		return nil, errors.Wrap(err, "could not save wallet to disk")
	}
	if err := w.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return nil, errors.Wrap(err, "could not write keymanager config to disk")
	}
	km, err := derived.KeymanagerForPhrase(ctx, &derived.SetupConfig{
		Opts:     derived.DefaultKeymanagerOpts(),
		Wallet:   w,
		Mnemonic: cfg.Mnemonic,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not make keymanager for given phrase")
	}
	if err := km.WriteEncryptedSeedToWallet(ctx, cfg.Mnemonic); err != nil {
		return nil, err
	}
	if cfg.NumAccounts == 1 {
		if _, err := km.CreateAccount(ctx, true /*logAccountInfo*/); err != nil {
			return nil, errors.Wrap(err, "could not create account in wallet")
		}
		return w, nil
	}
	for i := int64(0); i < cfg.NumAccounts; i++ {
		if _, err := km.CreateAccount(ctx, false /*logAccountInfo*/); err != nil {
			return nil, errors.Wrap(err, "could not create account in wallet")
		}
	}
	log.WithField("wallet-path", w.AccountsDir()).Infof(
		"Successfully recovered HD wallet with %d accounts. Please use accounts-v2 list to view details for your accounts",
		cfg.NumAccounts,
	)
	return w, nil
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
	allowedLanguages := map[string][]string{
		"english":             wordlists.English,
		"chinese_simplified":  wordlists.ChineseSimplified,
		"chinese_traditional": wordlists.ChineseTraditional,
		"french":              wordlists.French,
		"italian":             wordlists.Italian,
		"japanese":            wordlists.Japanese,
		"korean":              wordlists.Korean,
		"spanish":             wordlists.Spanish,
	}
	languages := make([]string, 0)
	for k := range allowedLanguages {
		languages = append(languages, k)
	}
	sort.Strings(languages)
	selectedLanguage, err := promptutil.ValidatePrompt(
		os.Stdin,
		fmt.Sprintf("Enter the language of your seed phrase: %s", strings.Join(languages, ", ")),
		func(input string) error {
			if _, ok := allowedLanguages[input]; !ok {
				return errors.New("input not in the list of allowed languages")
			}
			return nil
		},
	)
	if err != nil {
		return "", fmt.Errorf("could not get mnemonic language: %v", err)
	}
	bip39.SetWordList(allowedLanguages[selectedLanguage])
	mnemonicPhrase, err := promptutil.ValidatePrompt(
		os.Stdin,
		"Enter the seed phrase for the wallet you would like to recover",
		validateMnemonic)
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
