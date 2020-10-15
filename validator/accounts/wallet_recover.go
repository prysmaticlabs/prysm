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
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/promptutil"
	"github.com/prysmaticlabs/prysm/validator/accounts/prompt"
	"github.com/prysmaticlabs/prysm/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/prysmaticlabs/prysm/validator/keymanager/derived"
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
	w, _, err := RecoverWallet(cliCtx.Context, &RecoverWalletConfig{
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
			"Make a new validator account with ./prysm.sh validator accounts create",
	)
	return nil
}

// RecoverWallet uses a menmonic seed phrase to recover a wallet into the path provided.
func RecoverWallet(ctx context.Context, cfg *RecoverWalletConfig) (*wallet.Wallet, []*ethpb.Deposit_Data, error) {
	// Ensure that the wallet directory does not contain a wallet already
	dirExists, err := wallet.Exists(cfg.WalletDir)
	if err != nil {
		return nil, nil, err
	}
	if dirExists {
		return nil, nil, errors.New("a wallet already exists at this location. Please input an" +
			" alternative location for the new wallet or remove the current wallet")
	}
	w := wallet.New(&wallet.Config{
		WalletDir:      cfg.WalletDir,
		KeymanagerKind: keymanager.Derived,
		WalletPassword: cfg.WalletPassword,
	})
	keymanagerConfig, err := derived.MarshalOptionsFile(ctx, derived.DefaultKeymanagerOpts())
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not marshal keymanager config file")
	}
	if err := w.SaveWallet(); err != nil {
		return nil, nil, errors.Wrap(err, "could not save wallet to disk")
	}
	if err := w.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
		return nil, nil, errors.Wrap(err, "could not write keymanager config to disk")
	}
	km, err := derived.KeymanagerForPhrase(ctx, &derived.SetupConfig{
		Opts:     derived.DefaultKeymanagerOpts(),
		Wallet:   w,
		Mnemonic: cfg.Mnemonic,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not make keymanager for given phrase")
	}
	if err := km.WriteEncryptedSeedToWallet(ctx, cfg.Mnemonic); err != nil {
		return nil, nil, err
	}
	depositDataList := make([]*ethpb.Deposit_Data, cfg.NumAccounts)
	if cfg.NumAccounts == 1 {
		_, depositData, err := km.CreateAccount(ctx)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create account in wallet")
		}
		depositDataList[0] = depositData
		return w, nil, nil
	}
	for i := int64(0); i < cfg.NumAccounts; i++ {
		_, depositData, err := km.CreateAccount(ctx)
		if err != nil {
			return nil, nil, errors.Wrap(err, "could not create account in wallet")
		}
		depositDataList[i] = depositData
	}
	log.WithField("wallet-path", w.AccountsDir()).Infof(
		"Successfully recovered HD wallet with %d accounts. Please use accounts list to view details for your accounts",
		cfg.NumAccounts,
	)
	return w, depositDataList, nil
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
		return "", fmt.Errorf("could not get mnemonic language: %w", err)
	}
	bip39.SetWordList(allowedLanguages[selectedLanguage])
	mnemonicPhrase, err := promptutil.ValidatePrompt(
		os.Stdin,
		"Enter the seed phrase for the wallet you would like to recover",
		validateMnemonic)
	if err != nil {
		return "", fmt.Errorf("could not get mnemonic phrase: %w", err)
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
