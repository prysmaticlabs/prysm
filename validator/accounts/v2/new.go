package v2

import (
	"context"
	"errors"
	"os"
	"unicode"

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/validator/flags"
	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

const minPasswordLength = 8

var keymanagerKindSelections = map[v2keymanager.Kind]string{
	v2keymanager.Direct:  "Direct, On-Disk Accounts (Recommended)",
	v2keymanager.Derived: "Derived Accounts (Advanced)",
	v2keymanager.Remote:  "Remote Accounts (Advanced)",
}

// New creates a new validator account from user input. If a user
// does not have an initialized wallet at the specified wallet path, this
// method will create a new wallet and ask user for input for their new wallet's
// available options.
func New(cliCtx *cli.Context) error {
	// Read a wallet's directory from user input.
	walletDir := inputWalletDir(cliCtx)

	// Read the directory for password storage from user input.
	passwordsDirPath := inputPasswordsDirectory(cliCtx)

	ctx := context.Background()
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	var wallet *Wallet
	var err error
	var isNewWallet bool
	ok, err := hasWalletDir(walletDir)
	if err != nil {
		log.Fatalf("Could not check if wallet exists at %s: %v", walletDir, err)
	}
	if ok {
		// Read the wallet from the specified path.
		// Instantiate the wallet's keymanager from the wallet's
		// configuration file.
		wallet, err = ReadWallet(ctx, &WalletConfig{
			PasswordsDir: passwordsDirPath,
			WalletDir:    walletDir,
		})
		if err != nil {
			log.Fatalf("Could not read wallet at specified path %s: %v", walletDir, err)
		}
	} else {
		// Determine the desired wallet type from user input.
		keymanagerKind := inputKeymanagerKind(cliCtx)

		walletConfig := &WalletConfig{
			PasswordsDir:   passwordsDirPath,
			WalletDir:      walletDir,
			KeymanagerKind: keymanagerKind,
		}
		wallet, err = CreateWallet(ctx, walletConfig)
		if err != nil {
			log.Fatalf("Could not create wallet at specified path %s: %v", walletDir, err)
		}
		isNewWallet = true
	}

	// We initialize a new keymanager depending on the user's selected wallet type.
	keymanager := initializeWalletKeymanager(ctx, wallet, isNewWallet)

	// Read the new account's password from user input.
	password := inputAccountPassword(cliCtx)

	// Create a new validator account in the user's wallet.
	// TODO(#6220): Implement.
	if err := keymanager.CreateAccount(ctx, password); err != nil {
		log.Fatalf("Could not create account in wallet: %v", err)
	}
	return nil
}

func initializeWalletKeymanager(ctx context.Context, wallet *Wallet, isNewWallet bool) v2keymanager.IKeymanager {
	var keymanager v2keymanager.IKeymanager
	var err error
	if isNewWallet {
		switch wallet.KeymanagerKind() {
		case v2keymanager.Direct:
			keymanager = direct.NewKeymanager(ctx, wallet, direct.DefaultConfig())
		case v2keymanager.Derived:
			log.Fatal("Derived keymanagers are unimplemented, work in progress")
		case v2keymanager.Remote:
			log.Fatal("Remote keymanagers are unimplemented, work in progress")
		default:
			log.Fatal("Keymanager type must be specified")
		}
		keymanagerConfig, err := keymanager.ConfigFile(ctx)
		if err != nil {
			log.Fatalf("Could not marshal keymanager config file: %v", err)
		}
		if err := wallet.WriteKeymanagerConfigToDisk(ctx, keymanagerConfig); err != nil {
			log.Fatalf("Could not write keymanager config file to disk: %v", err)
		}
		return keymanager
	}
	switch wallet.KeymanagerKind() {
	case v2keymanager.Direct:
		keymanager, err = direct.NewKeymanagerFromConfigFile(ctx, wallet)
		if err != nil {
			log.Fatal(err)
		}
	case v2keymanager.Derived:
		log.Fatal("Derived keymanagers are unimplemented, work in progress")
	case v2keymanager.Remote:
		log.Fatal("Remote keymanagers are unimplemented, work in progress")
	default:
		log.Fatal("Keymanager kind must be specified")
	}
	return keymanager
}

// Check if a user has an existing wallet at the specified path.
func hasWalletDir(walletPath string) (bool, error) {
	_, err := os.Stat(walletPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func inputWalletDir(cliCtx *cli.Context) string {
	datadir := cliCtx.String(flags.WalletDirFlag.Name)
	prompt := promptui.Prompt{
		Label:    "Enter a wallet directory",
		Validate: validateDirectoryPath,
		Default:  datadir,
	}
	walletPath, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not determine wallet directory: %v", formatPromptError(err))
	}
	return walletPath
}

func inputKeymanagerKind(_ *cli.Context) v2keymanager.Kind {
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
		log.Fatalf("Could not select wallet type: %v", formatPromptError(err))
	}
	return v2keymanager.Kind(selection)
}

func inputAccountPassword(_ *cli.Context) string {
	prompt := promptui.Prompt{
		Label:    "New account password",
		Validate: validatePasswordInput,
		Mask:     '*',
	}

	walletPassword, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not read wallet password: %v", formatPromptError(err))
	}

	prompt = promptui.Prompt{
		Label: "Confirm password",
		Mask:  '*',
	}
	confirmPassword, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not read password confirmation: %v", formatPromptError(err))
	}
	if walletPassword != confirmPassword {
		log.Fatal("Passwords do not match")
	}
	return walletPassword
}

func inputPasswordsDirectory(cliCtx *cli.Context) string {
	passwordsDir := cliCtx.String(flags.WalletPasswordsDirFlag.Name)
	prompt := promptui.Prompt{
		Label:    "Passwords directory",
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
