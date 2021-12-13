package accounts

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/cmd/validator/flags"
	"github.com/prysmaticlabs/prysm/io/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/userprompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
	"github.com/tyler-smith/go-bip39"
	"github.com/tyler-smith/go-bip39/wordlists"
	"github.com/urfave/cli/v2"
)

const (
	phraseWordCount = 24
	// #nosec G101 -- Not sensitive data
	newMnemonicPassphraseYesNoText = "(Advanced) Do you want to setup a '25th word' passphrase for your mnemonic? [y/n]"
	// #nosec G101 -- Not sensitive data
	newMnemonicPassphrasePromptText = "(Advanced) Setup a passphrase '25th word' for your mnemonic " +
		"(WARNING: You cannot recover your keys from your mnemonic if you forget this passphrase!)"
	// #nosec G101 -- Not sensitive data
	mnemonicPassphraseYesNoText = "(Advanced) Do you have an optional '25th word' passphrase for your mnemonic? [y/n]"
	// #nosec G101 -- Not sensitive data
	mnemonicPassphrasePromptText = "(Advanced) Enter the '25th word' passphrase for your mnemonic"
)

// RecoverWalletConfig to run the recover wallet function.
type RecoverWalletConfig struct {
	WalletDir        string
	WalletPassword   string
	Mnemonic         string
	NumAccounts      int
	Mnemonic25thWord string
}

// RecoverWalletCli uses a menmonic seed phrase to recover a wallet into the path provided. This
// uses the CLI to extract necessary values to run the function.
func RecoverWalletCli(cliCtx *cli.Context) error {
	mnemonic, err := inputMnemonic(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get mnemonic phrase")
	}
	config := &RecoverWalletConfig{
		Mnemonic: mnemonic,
	}
	skipMnemonic25thWord := cliCtx.IsSet(flags.SkipMnemonic25thWordCheckFlag.Name)
	has25thWordFile := cliCtx.IsSet(flags.Mnemonic25thWordFileFlag.Name)
	if !skipMnemonic25thWord && !has25thWordFile {
		resp, err := prompt.ValidatePrompt(
			os.Stdin, mnemonicPassphraseYesNoText, prompt.ValidateYesOrNo,
		)
		if err != nil {
			return errors.Wrap(err, "could not validate choice")
		}
		if strings.EqualFold(resp, "y") {
			mnemonicPassphrase, err := prompt.InputPassword(
				cliCtx,
				flags.Mnemonic25thWordFileFlag,
				mnemonicPassphrasePromptText,
				"Confirm mnemonic passphrase",
				false, /* Should confirm password */
				func(input string) error {
					if strings.TrimSpace(input) == "" {
						return errors.New("input cannot be empty")
					}
					return nil
				},
			)
			if err != nil {
				return err
			}
			config.Mnemonic25thWord = mnemonicPassphrase
		}
	}
	walletDir, err := userprompt.InputDirectory(cliCtx, userprompt.WalletDirPromptText, flags.WalletDirFlag)
	if err != nil {
		return err
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
		return err
	}
	numAccounts, err := inputNumAccounts(cliCtx)
	if err != nil {
		return errors.Wrap(err, "could not get number of accounts to recover")
	}
	config.WalletDir = walletDir
	config.WalletPassword = walletPassword
	config.NumAccounts = int(numAccounts)
	if _, err = RecoverWallet(cliCtx.Context, config); err != nil {
		return err
	}
	log.Infof(
		"Successfully recovered HD wallet with accounts and saved configuration to disk",
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
		KeymanagerKind: keymanager.Derived,
		WalletPassword: cfg.WalletPassword,
	})
	if err := w.SaveWallet(); err != nil {
		return nil, errors.Wrap(err, "could not save wallet to disk")
	}
	km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
		Wallet:           w,
		ListenForChanges: false,
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not make keymanager for given phrase")
	}
	if err := km.RecoverAccountsFromMnemonic(ctx, cfg.Mnemonic, cfg.Mnemonic25thWord, cfg.NumAccounts); err != nil {
		return nil, err
	}
	log.WithField("wallet-path", w.AccountsDir()).Infof(
		"Successfully recovered HD wallet with %d accounts. Please use `accounts list` to view details for your accounts",
		cfg.NumAccounts,
	)
	return w, nil
}

func inputMnemonic(cliCtx *cli.Context) (mnemonicPhrase string, err error) {
	if cliCtx.IsSet(flags.MnemonicFileFlag.Name) {
		mnemonicFilePath := cliCtx.String(flags.MnemonicFileFlag.Name)
		data, err := ioutil.ReadFile(mnemonicFilePath) // #nosec G304 -- ReadFile is safe
		if err != nil {
			return "", err
		}
		enteredMnemonic := string(data)
		if err := ValidateMnemonic(enteredMnemonic); err != nil {
			return "", errors.Wrap(err, "mnemonic phrase did not pass validation")
		}
		return enteredMnemonic, nil
	}
	allowedLanguages := map[string][]string{
		"chinese_simplified":  wordlists.ChineseSimplified,
		"chinese_traditional": wordlists.ChineseTraditional,
		"czech":               wordlists.Czech,
		"english":             wordlists.English,
		"french":              wordlists.French,
		"japanese":            wordlists.Japanese,
		"korean":              wordlists.Korean,
		"italian":             wordlists.Italian,
		"spanish":             wordlists.Spanish,
	}
	languages := make([]string, 0)
	for k := range allowedLanguages {
		languages = append(languages, k)
	}
	sort.Strings(languages)
	selectedLanguage, err := prompt.ValidatePrompt(
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
		return "", fmt.Errorf("could not get mnemonic language: %w", err)
	}
	bip39.SetWordList(allowedLanguages[selectedLanguage])
	mnemonicPhrase, err = prompt.ValidatePrompt(
		os.Stdin,
		"Enter the seed phrase for the wallet you would like to recover",
		ValidateMnemonic)
	if err != nil {
		return "", fmt.Errorf("could not get mnemonic phrase: %w", err)
	}
	return mnemonicPhrase, nil
}

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

// ValidateMnemonic ensures that it is not empty and that the count of the words are
// as specified(currently 24).
func ValidateMnemonic(mnemonic string) error {
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
