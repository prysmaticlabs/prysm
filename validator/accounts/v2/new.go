package v2

import (
	"errors"
	"unicode"

	"github.com/manifoldco/promptui"
	logrus "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/prysmaticlabs/prysm/shared/cmd"
)

var log = logrus.WithField("prefix", "accounts-v2")

// Steps: ask for the path to store the validator datadir
// Ask for the type of wallet: direct, derived, remote
// Ask for where to store passwords: default path within datadir
// Ask them to enter a password for the account.
// If user already has account, instead just make a new account with a good name
// Allow for creating more than 1 validator??
// TODOS: mnemonic for withdrawal key, ensure they write it down.
func New(cliCtx *cli.Context) error {
	datadir := cliCtx.String(cmd.DataDirFlag.Name)
	prompt := promptui.Prompt{
		Label:    "Enter a wallet directory",
		Validate: validateDirectoryPath,
		Default:  datadir,
	}
	walletPath, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not determine wallet directory: %v", formatPromptError(err))
	}
	_ = walletPath

	promptSelect := promptui.Select{
		Label: "Select a type of wallet",
		Items: []string{
			"Direct, On-Disk (Recommended)",
			"Derived (Advanced)",
			"Remote (Advanced)",
		},
	}
	_, walletType, err := promptSelect.Run()
	if err != nil {
		log.Fatalf("Could not select wallet type: %v", formatPromptError(err))
	}
	_ = walletType

	prompt = promptui.Prompt{
		Label:    "Strong password",
		Validate: validatePasswordInput,
		Mask:     '*',
	}

	walletPassword, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not read wallet password: %v", formatPromptError(err))
	}
	_ = walletPassword

	prompt = promptui.Prompt{
		Label:    "Enter the directory where passwords will be stored",
		Validate: validateDirectoryPath,
		Default:  datadir,
	}
	passwordsPath, err := prompt.Run()
	if err != nil {
		log.Fatalf("Could not determine passwords directory: %v", formatPromptError(err))
	}
	_ = passwordsPath
	return nil
}

func validatePasswordInput(input string) error {
	var (
		hasMinLen  = false
		hasLetter  = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(input) >= 8 {
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
