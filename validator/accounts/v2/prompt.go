package v2

import (
	"fmt"
	"io/ioutil"
	"unicode"

	"github.com/manifoldco/promptui"
	strongPasswords "github.com/nbutton23/zxcvbn-go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/validator/flags"
	"github.com/urfave/cli/v2"
)

const (
	importDirPromptText          = "Enter the file location of the exported wallet zip to import"
	exportDirPromptText          = "Enter a file location to write the exported wallet to"
	walletDirPromptText          = "Enter a wallet directory"
	passwordsDirPromptText       = "Directory where passwords will be stored"
	newWalletPasswordPromptText  = "New wallet password"
	confirmPasswordPromptText    = "Confirm password"
	walletPasswordPromptText     = "Wallet password"
	newAccountPasswordPromptText = "New account password"
	passwordForAccountPromptText = "Enter password for account %s"
)

type passwordConfirm int

const (
	// Constants for passwords.
	minPasswordLength = 8
	// Min password score of 3 out of 5 based on the https://github.com/nbutton23/zxcvbn-go
	// library for strong-entropy password computation.
	minPasswordScore = 3

	noConfirmPass passwordConfirm = iota
	confirmPass
)

func inputDir(cliCtx *cli.Context, promptText string, flag *cli.StringFlag) (string, error) {
	directory := cliCtx.String(flag.Name)
	if cliCtx.IsSet(flag.Name) {
		return directory, nil
	}

	prompt := promptui.Prompt{
		Label:    promptText,
		Validate: validateDirectoryPath,
		Default:  directory,
	}
	inputtedDir, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("could not determine directory: %v", formatPromptError(err))
	}
	return inputtedDir, nil
}

func inputPassword(cliCtx *cli.Context, promptText string, confirmPassword passwordConfirm) (string, error) {
	if cliCtx.IsSet(flags.PasswordFileFlag.Name) {
		passwordFilePath := cliCtx.String(flags.PasswordFileFlag.Name)
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", err
		}
		enteredPassword := string(data)
		if err := validatePasswordInput(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return enteredPassword, nil
	}

	var hasValidPassword bool
	var walletPassword string
	var err error
	for !hasValidPassword {
		prompt := promptui.Prompt{
			Label:    promptText,
			Validate: validatePasswordInput,
			Mask:     '*',
		}

		walletPassword, err = prompt.Run()
		if err != nil {
			return "", fmt.Errorf("could not read account password: %v", formatPromptError(err))
		}

		if confirmPassword == confirmPass {
			prompt = promptui.Prompt{
				Label: confirmPasswordPromptText,
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
		} else {
			return walletPassword, nil
		}
	}
	return walletPassword, nil
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
