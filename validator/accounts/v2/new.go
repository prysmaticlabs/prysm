package v2

import (
	"context"
	"errors"
	"io"
	"os"
	"unicode"

	"github.com/manifoldco/promptui"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/prysmaticlabs/prysm/validator/keymanager/v2/direct"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = logrus.WithField("prefix", "accounts-v2")

// WalletType defines an enum for either direct, derived, or remote-signing
// wallets as specified by a user during account creation.
type WalletType int

const (
	directWallet  WalletType = iota // Direct, on-disk wallet.
	derivedWallet                   // Derived, hierarchical-deterministic wallet.
	remoteWallet                    // Remote-signing wallet.
)

const minPasswordLength = 8

var walletTypeSelections = map[WalletType]string{
	directWallet:  "Direct, On-Disk Accounts (Recommended)",
	derivedWallet: "Derived Accounts (Advanced)",
	remoteWallet:  "Remote Accounts (Advanced)",
}

// New --
func New(cliCtx *cli.Context) error {
	// Read a wallet path and the desired type of wallet for a user
	// (e.g.: Direct, Keystore, Derived).
	walletPath := inputWalletPath(cliCtx)

	ctx := context.Background()
	// Check if the user has a wallet at the specified path.
	// If a user does not have a wallet, we instantiate one
	// based on specified options.
	var wallet *Wallet
	var err error
	ok, err := hasWalletDir(walletPath)
	if err != nil {
		log.Fatalf("Could not check if wallet exists at %s: %v", walletPath, err)
	}
	if ok {
		// Read the wallet from the specified path.
		// Instantiate the wallet's keymanager from the wallet's
		// configuration file.
		wallet, err = ReadWallet(ctx, walletPath)
		if err != nil {
			log.Fatalf("Could not read wallet at specified path %s: %v", walletPath, err)
		}
	} else {
		// We create a new account for the user given a wallet.
		walletType := inputWalletType(cliCtx)

		// Read the directory for password storage from user input.
		passwordsDirPath := inputPasswordsDirectory(cliCtx)

		// Open the wallet and password directories for writing.
		walletConfig := &WalletConfig{
			PasswordsDir: passwordsDirPath,
			WalletDir:    walletPath,
		}
		switch walletType {
		case directWallet:
			directKeymanager := direct.NewKeymanager(ctx, direct.DefaultConfig())
			walletConfig.Keymanager = directKeymanager
		case derivedWallet:
			log.Fatal("Derived wallets are unimplemented, work in progress")
		case remoteWallet:
			log.Fatal("Remote wallets are unimplemented, work in progress")
		}
		wallet, err = CreateWallet(ctx, walletConfig)
		if err != nil {
			log.Fatalf("Could not create wallet at specified path %s: %v", walletPath, err)
		}
	}

	// Read the account password from user input.
	password := inputAccountPassword(cliCtx)

	// Create a new validator account in the user's wallet.
	// TODO(#6220): Implement by utilizing the wallet.keymanager.CreateAccount()
	// method accordingly.
	_ = wallet
	_ = password
	return nil
}

// Check if a user has an existing wallet at the specified path.
func hasWalletDir(walletPath string) (bool, error) {
	f, err := os.Open(walletPath)
	if err != nil {
		return false, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	// Either not empty or error, suits both cases.
	return false, err
}

func inputWalletPath(cliCtx *cli.Context) string {
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

func inputWalletType(_ *cli.Context) WalletType {
	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			walletTypeSelections[directWallet],
			walletTypeSelections[derivedWallet],
			walletTypeSelections[remoteWallet],
		},
	}
	selection, _, err := promptSelect.Run()
	if err != nil {
		log.Fatalf("Could not select wallet type: %v", formatPromptError(err))
	}
	return WalletType(selection)
}

func inputAccountPassword(_ *cli.Context) string {
	prompt := promptui.Prompt{
		Label:    "Strong password",
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
		Label:    "Enter the directory where passwords will be stored",
		Validate: validateDirectoryPath,
		Default:  passwordsDir,
	}
	passwordsPath, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not determine passwords directory: %v", formatPromptError(err))
	}
	return passwordsPath
}

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
