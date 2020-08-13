package promptutil

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/fileutil"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

var au = aurora.NewAurora(true)

// PasswordConfirm defines an enum type that can determine whether or not
// a prompt should confirm a password input.
type PasswordConfirm int

const (
	// NoConfirmPass enum to indicate to the prompt that confirming the password is not needed.
	NoConfirmPass PasswordConfirm = iota
	// ConfirmPass enum to indicate to the prompt to confirm the password entered.
	ConfirmPass
)

// ValidatePrompt requests the user for text and expects it to fulfil la provided validation function.
func ValidatePrompt(promptText string, validateFunc func(string) error) (string, error) {
	var responseValid bool
	var response string
	for !responseValid {
		fmt.Printf("%s:\n", au.Bold(promptText))
		scanner := bufio.NewScanner(os.Stdin)
		if ok := scanner.Scan(); ok {
			item := scanner.Text()
			response = strings.TrimRight(item, "\r\n")
			if err := validateFunc(response); err != nil {
				fmt.Printf("Entry not valid: %s\n", au.BrightRed(err))
			} else {
				responseValid = true
			}
		} else {
			return "", errors.New("could not scan text input")
		}
	}
	return response, nil
}

// DefaultPrompt prompts the user for any text and performs no validation. If nothing is entered it returns the default.
func DefaultPrompt(promptText string, defaultValue string) (string, error) {
	var response string
	if defaultValue != "" {
		fmt.Printf("%s %s:\n", promptText, fmt.Sprintf("(%s: %s)", au.BrightGreen("default"), defaultValue))
	} else {
		fmt.Printf("%s\n", promptText)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if ok := scanner.Scan(); ok {
		item := scanner.Text()
		response = strings.TrimRight(item, "\r\n")
		if response == "" {
			return defaultValue, nil
		}
		return response, nil
	}
	return "", errors.New("could not scan text input")
}

// DefaultAndValidatePrompt prompts the user for any text and expects it to fulfill a validation function. If nothing is entered
// the default value is returned.
func DefaultAndValidatePrompt(promptText string, defaultValue string, validateFunc func(string) error) (string, error) {
	var responseValid bool
	var response string
	for !responseValid {
		fmt.Printf("%s %s:\n", promptText, fmt.Sprintf("(%s: %s)", au.BrightGreen("default"), defaultValue))
		scanner := bufio.NewScanner(os.Stdin)
		if ok := scanner.Scan(); ok {
			item := scanner.Text()
			response = strings.TrimRight(item, "\r\n")
			if err := validateFunc(response); err != nil {
				fmt.Printf("Entry not valid: %s\n", au.BrightRed(err))
			} else {
				responseValid = true
			}
		} else {
			return "", errors.New("could not scan text input")
		}
	}
	return response, nil
}

// PasswordPrompt prompts the user for a password, that repeatedly requests the password until it qualifies the
// passed in validation function.
func PasswordPrompt(promptText string, validateFunc func(string) error) (string, error) {
	var responseValid bool
	var response string
	for !responseValid {
		fmt.Printf("%s: ", au.Bold(promptText))
		bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		response = strings.TrimRight(string(bytePassword), "\r\n")
		if err := validateFunc(response); err != nil {
			fmt.Printf("\nEntry not valid: %s\n", au.BrightRed(err))
		} else {
			fmt.Println("")
			responseValid = true
		}
	}
	return response, nil
}

// InputPassword with a custom validator along capabilities of confirming
// the password and reading it from disk if a specified flag is set.
func InputPassword(
	cliCtx *cli.Context,
	passwordFileFlag *cli.StringFlag,
	promptText string,
	confirmText string,
	shouldConfirmPassword PasswordConfirm,
	passwordValidator func(input string) error,
) (string, error) {
	if cliCtx.IsSet(passwordFileFlag.Name) {
		passwordFilePathInput := cliCtx.String(passwordFileFlag.Name)
		passwordFilePath, err := fileutil.ExpandPath(passwordFilePathInput)
		if err != nil {
			return "", errors.Wrap(err, "could not determine absolute path of password file")
		}
		data, err := ioutil.ReadFile(passwordFilePath)
		if err != nil {
			return "", errors.Wrap(err, "could not read password file")
		}
		enteredPassword := strings.TrimRight(string(data), "\r\n")
		if err := passwordValidator(enteredPassword); err != nil {
			return "", errors.Wrap(err, "password did not pass validation")
		}
		return enteredPassword, nil
	}
	var hasValidPassword bool
	var password string
	var err error
	for !hasValidPassword {
		password, err = PasswordPrompt(promptText, passwordValidator)
		if err != nil {
			return "", fmt.Errorf("could not read password: %v", err)
		}
		if shouldConfirmPassword == ConfirmPass {
			passwordConfirmation, err := PasswordPrompt(confirmText, passwordValidator)
			if err != nil {
				return "", fmt.Errorf("could not read password confirmation: %v", err)
			}
			if password != passwordConfirmation {
				log.Error("Passwords do not match")
				continue
			}
			hasValidPassword = true
		} else {
			return password, nil
		}
	}
	return password, nil
}
