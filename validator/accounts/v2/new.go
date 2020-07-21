package v2

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"unicode"

	"github.com/manifoldco/promptui"
	strongPasswords "github.com/nbutton23/zxcvbn-go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/derived"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

const (
	minPasswordLength = 8
	// Min password score of 3 out of 5 based on the https://github.com/nbutton23/zxcvbn-go
	// library for strong-entropy password computation.
	minPasswordScore = 3
)

var keymanagerKindSelections = map[v2keymanager.Kind]string{
	v2keymanager.Derived: "HD Wallet (Recommended)",
	v2keymanager.Direct:  "Non-HD Wallet (Most Basic)",
	v2keymanager.Remote:  "Remote Signing Wallet (Advanced)",
}

// NewAccount creates a new validator account from user input by opening
// a wallet from the user's specified path.
func NewAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if errors.Is(err, ErrNoWalletFound) {
		log.Fatal("No wallet found, create a new one with ./prysm.sh validator wallet-v2 create")
	} else if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
	if err != nil {
		log.Fatal(err)
	}
	skipMnemonicConfirm := cliCtx.Bool(flags.SkipMnemonicConfirmFlag.Name)
	switch keymanagerKind {
	case v2keymanager.Remote:
		log.Fatal("Cannot create a new account for a remote keymanager")
	case v2keymanager.Direct:
		passwordsDirPath := inputPasswordsDirectory(cliCtx)
		// Read the directory for password storage from user input.
		wallet, err := OpenWallet(ctx, &WalletConfig{
			PasswordsDir:      passwordsDirPath,
			WalletDir:         walletDir,
			CanUnlockAccounts: true,
		})
		if err != nil {
			log.Fatalf("Could not open wallet: %v", err)
		}
		configFile, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := direct.UnmarshalConfigFile(configFile)
		if err != nil {
			log.Fatal(err)
		}
		keymanager, err := direct.NewKeymanager(ctx, wallet, cfg, skipMnemonicConfirm)
		if err != nil {
			log.Fatal(err)
		}
		password, err := inputNewAccountPassword(cliCtx)
		if err != nil {
			log.Fatal(err)
		}
		// Create a new validator account using the specified keymanager.
		if _, err := keymanager.CreateAccount(ctx, password); err != nil {
			log.Fatalf("Could not create account in wallet: %v", err)
		}
	case v2keymanager.Derived:
		// Read the directory for password storage from user input.
		wallet, err := OpenWallet(ctx, &WalletConfig{
			WalletDir:         walletDir,
			CanUnlockAccounts: true,
		})
		if err != nil {
			log.Fatalf("Could not open wallet: %v", err)
		}
		configFile, err := wallet.ReadKeymanagerConfigFromDisk(ctx)
		if err != nil {
			log.Fatal(err)
		}
		cfg, err := derived.UnmarshalConfigFile(configFile)
		if err != nil {
			log.Fatal(err)
		}
		walletPassword, err := inputExistingWalletPassword()
		if err != nil {
			log.Fatal(err)
		}
		keymanager, err := derived.NewKeymanager(ctx, wallet, cfg, skipMnemonicConfirm, walletPassword)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := keymanager.CreateAccount(ctx); err != nil {
			log.Fatalf("Could not create account in wallet: %v", err)
		}
	default:
		log.Fatal("Keymanager kind not supported")
	}
	return nil
}

func inputWalletDir(cliCtx *cli.Context) (string, error) {
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
	if cliCtx.IsSet(flags.WalletDirFlag.Name) {
		return walletDir, nil
	}

	if walletDir == flags.DefaultValidatorDir() {
		walletDir = path.Join(walletDir, WalletDefaultDirName)
	}
	prompt := promptui.Prompt{
		Label:    "Enter a wallet directory",
		Validate: validateDirectoryPath,
		Default:  walletDir,
	}
	walletPath, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine wallet directory: %v", formatPromptError(err))
	}
	ok, err := hasDir(walletPath)
	if err != nil {
		return "", errors.Wrapf(err, "could not check if wallet dir %s exists", walletDir)
	}
	if !ok {
		return walletPath, ErrNoWalletFound
	}
	return walletPath, nil
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

func inputNewWalletPassword() (string, error) {
	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		prompt := promptui.Prompt{
			Label:    "New wallet password",
			Validate: validatePasswordInput,
			Mask:     '*',
		}

		walletPassword, err = prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read wallet password: %v", formatPromptError(err))
		}

		prompt = promptui.Prompt{
			Label: "Confirm password",
			Mask:  '*',
		}
		confirmPassword, err := prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read password confirmation: %v", formatPromptError(err))
		}
		if walletPassword != confirmPassword {
			log.Error("Passwords do not match")
			continue
		}
		hasValidPassword = true
	}
	return walletPassword, nil
}

func inputExistingWalletPassword() (string, error) {
	prompt := promptui.Prompt{
		Label:    "Wallet password",
		Validate: validatePasswordInput,
		Mask:     '*',
	}

	walletPassword, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not read wallet password: %v", formatPromptError(err))
	}
	return walletPassword, nil
}

func inputNewAccountPassword(cliCtx *cli.Context) (string, error) {
	if cliCtx.IsSet(flags.PasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.PasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", err
		}
		enteredPassword := string(data)
		if err := validatePasswordInput(enteredPassword); err != nil {
			log.WithError(err).Fatal("Password did not pass validation")
		}
		return enteredPassword, nil
	}

	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		prompt := promptui.Prompt{
			Label:    "New account password",
			Validate: validatePasswordInput,
			Mask:     '*',
		}

		walletPassword, err = prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", formatPromptError(err))
		}

		prompt = promptui.Prompt{
			Label: "Confirm password",
			Mask:  '*',
		}
		confirmPassword, err := prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read password confirmation: %v", formatPromptError(err))
		}
		if walletPassword != confirmPassword {
			log.Error("Passwords do not match")
			continue
		}
		hasValidPassword = true
	}
	return walletPassword, nil
}

func inputPasswordForAccount(_ *cli.Context, accountName string) (string, error) {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Enter password for account %s", accountName),
		Mask:  '*',
	}

	walletPassword, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not read wallet password: %v", formatPromptError(err))
	}
	return walletPassword, nil
}

func inputPasswordsDirectory(cliCtx *cli.Context) string {
	passwordsDir := cliCtx.String(flags.WalletPasswordsDirFlag.Name)
	if cliCtx.IsSet(flags.WalletPasswordsDirFlag.Name) {
		return passwordsDir
	}

	if passwordsDir == flags.DefaultValidatorDir() {
		passwordsDir = path.Join(passwordsDir, PasswordsDefaultDirName)
	}
	prompt := promptui.Prompt{
		Label:    "Directory where passwords will be stored",
		Validate: validateDirectoryPath,
		Default:  passwordsDir,
	}
	passwordsPath, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not determine passwords directory: %v", formatPromptError(err))
	}
	return passwordsPath
}

// Validate a strong password input for new accounts,
// including a min length, at least 1 number and at least
// 1 special character.
func validatePasswordInput(input string) error {
	var (
		hasMinLen  = false
		hasLetter  = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(input) >= minPasswordLength {
		hasMinLen = true
	}
	for _, char := range input {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	if !(hasMinLen && hasLetter && hasNumber && hasSpecial) {
		return errors.New(
			"password must have more than 8 characters, at least 1 special character, and 1 number",
		)
	}
	strength := strongPasswords.PasswordStrength(input, nil)
	if strength.Score < minPasswordScore {
		return errors.New(
			"password is too easy to guess, try a stronger password",
		)
	}
	return nil
}

func validateDirectoryPath(input string) error {
	if len(input) == 0 {
		return errors.New("directory path must not be empty")
	}
	return nil
}

func formatPromptError(err error) error {
	switch err {
	case promptui.ErrAbort:
		return errors.New("wallet creation aborted, closing")
	case promptui.ErrInterrupt:
		return errors.New("keyboard interrupt, closing")
	case promptui.ErrEOF:
		return errors.New("no input received, closing")
	default:
		return err
	}
}
