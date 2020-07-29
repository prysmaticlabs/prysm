package promptutil

import (
	"fmt"
	"os"
	"strings"

	"github.com/logrusorgru/aurora"
	"golang.org/x/crypto/ssh/terminal"
)

var au = aurora.NewAurora(true)

// ValidatePrompt requests the user for text and expects it to fulfil la provided validation function.
func ValidatePrompt(promptText string, validateFunc func(string) error) (string, error) {
	var responseValid bool
	var response string
	for !responseValid {
		fmt.Printf("%s:\n", au.Bold(promptText))
		_, err := fmt.Scanln(&response)
		if err != nil && !strings.Contains(err.Error(), "unexpected newline") {
			return "", err
		}
		response = strings.TrimRight(response, "\r\n")
		if err := validateFunc(response); err != nil {
			fmt.Printf("Entry not valid: %s\n", au.BrightRed(err))
		} else {
			responseValid = true
		}
	}
	return response, nil
}

// DefaultPrompt prompts the user for any text and performs no validation. If nothing is entered it returns the default.
func DefaultPrompt(promptText string, defaultValue string) (string, error) {
	var response string
	fmt.Printf("%s %s:\n", promptText, fmt.Sprintf("(%s: %s)", au.BrightGreen("default"), defaultValue))
	_, err := fmt.Scanln(&response)
	if err != nil && !strings.Contains(err.Error(), "unexpected newline") {
		return "", err
	}
	response = strings.TrimRight(response, "\r\n")
	if response == "" {
		return defaultValue, nil
	}
	return response, nil
}

// DefaultAndValidatePrompt prompts the user for any text and expects it to fulfill a validation function. If nothing is entered
// the default value is returned.
func DefaultAndValidatePrompt(promptText string, defaultValue string, validateFunc func(string) error) (string, error) {
	var responseValid bool
	var response string
	for !responseValid {
		fmt.Printf("%s %s:\n", promptText, fmt.Sprintf("(%s: %s)", au.BrightGreen("default"), defaultValue))
		_, err := fmt.Scanln(&response)
		if err != nil && !strings.Contains(err.Error(), "unexpected newline") {
			return "", err
		}
		response = strings.TrimRight(response, "\r\n")
		if response == "" {
			return defaultValue, nil
		}
		if err := validateFunc(response); err != nil {
			fmt.Printf("Entry not valid: %s\n", au.BrightRed(err))
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
		fmt.Printf("%s:\n", au.Bold(promptText))
		bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		response = strings.TrimRight(string(bytePassword), "\r\n")
		if err := validateFunc(response); err != nil {
			fmt.Printf("\nEntry not valid: %s\n", au.BrightRed(err))
		} else {
			responseValid = true
		}
	}
	return response, nil
}
