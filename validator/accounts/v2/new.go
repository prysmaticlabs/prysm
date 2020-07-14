package v2

import (
	"context"
	"fmt"
	"path"
	"unicode"

	"github.com/manifoldco/promptui"
	strongPasswords "github.com/nbutton23/zxcvbn-go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
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
	v2keymanager.Direct:  "Direct, On-Disk Wallet (Recommended)",
	v2keymanager.Derived: "Derived HD Wallet (Advanced)",
	v2keymanager.Remote:  "Remote Signing Wallet (Advanced)",
}

// NewAccount creates a new validator account from user input. If a user
// does not have an initialized wallet at the specified wallet path, this
// method will create a new wallet and ask user for input for their new wallet's
// available options.
func NewAccount(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir, err := inputWalletDir(cliCtx)
	if err != nil {
		log.Fatalf("Could not parse wallet directory: %v", err)
	}
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	walletExists, err := hasDir(walletDir)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	var wallet *Wallet
	if walletExists {
		keymanagerKind, err := readKeymanagerKindFromWalletPath(walletDir)
		if err != nil {
			log.Fatal(err)
		}
		if keymanagerKind == v2keymanager.Remote {
			log.Fatal("Cannot create an account for a remote keymanager")
		}
		// Read the directory for password storage from user input.
		passwordsDirPath := inputPasswordsDirectory(cliCtx)
		wallet, err = OpenWallet(ctx, &WalletConfig{
			PasswordsDir:      passwordsDirPath,
			WalletDir:         walletDir,
			CanUnlockAccounts: true,
			KeymanagerKind:    keymanagerKind,
		})
	} else {
		// Determine the desired keymanager kind for the wallet from user input.
		if err := CreateWallet(cliCtx, walletDir); err != nil {
			log.Fatalf("Could not create new wallet: %v", err)
		}
	}

	if wallet.KeymanagerKind() != v2keymanager.Direct {
		log.Fatalf("cannot create a new account for a %s keymanager", wallet.KeymanagerKind())
	}

	// We initialize a new keymanager depending on the wallet's keymanager kind.
	keymanager, err := wallet.InitializeKeymanager(ctx)
	if err != nil {
		log.Fatalf("Could not initialize keymanager: %v", err)
	}

	// Read the new account's password from user input.
	password, err := inputNewAccountPassword(cliCtx)
	if err != nil {
		log.Fatalf("Could not read password: %v", err)
	}

	// Create a new validator account using the specified keymanager.
	if _, err := keymanager.CreateAccount(ctx, password); err != nil {
		log.Fatalf("Could not create account in wallet: %v", err)
	}
	return nil
}

func inputWalletDir(cliCtx *cli.Context) (string, error) {
	walletDir := cliCtx.String(flags.WalletDirFlag.Name)
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
	return walletPath, nil
}

func inputKeymanagerKind(_ *cli.Context) (v2keymanager.Kind, error) {
	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			keymanagerKindSelections[v2keymanager.Direct],
			keymanagerKindSelections[v2keymanager.Derived],
			keymanagerKindSelections[v2keymanager.Remote],
		},
	}
	selection, _, err := promptSelect.Run()
	if err != nil {
		return v2keymanager.Direct, fmt.Errorf("could not select wallet type: %v", formatPromptError(err))
	}
	return v2keymanager.Kind(selection), nil
}

func inputNewAccountPassword(_ *cli.Context) (string, error) {
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
