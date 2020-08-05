package promptutil

import (
	"errors"
	"strconv"
	"unicode"

	strongPasswords "github.com/nbutton23/zxcvbn-go"
)

const (
	// Constants for passwords.
	minPasswordLength = 8
	// Min password score of 2 out of 5 based on the https://github.com/nbutton23/zxcvbn-go
	// library for strong-entropy password computation.
	minPasswordScore = 2
)

// NotEmpty is a validation function to make sure the input given isn't empty and is valid unicode.
func NotEmpty(input string) error {
	if input == "" {
		return errors.New("input cannot be empty")
	}
	if !IsValidUnicode(input) {
		return errors.New("not valid unicode")
	}
	return nil
}

// ValidateNumber makes sure the entered text is a valid number.
func ValidateNumber(input string) error {
	_, err := strconv.Atoi(input)
	if err != nil {
		return err
	}
	return nil
}

// ValidateConfirmation makes sure the entered text is the user confirming.
func ValidateConfirmation(input string) error {
	if input != "Y" && input != "y" {
		return errors.New("please confirm the above text")
	}
	return nil
}

// IsValidUnicode checks if an input string is a valid unicode string comprised of only
// letters, numbers, punctuation, or symbols.
func IsValidUnicode(input string) bool {
	for _, char := range input {
		if !(unicode.IsLetter(char) ||
			unicode.IsNumber(char) ||
			unicode.IsPunct(char) ||
			unicode.IsSymbol(char) ||
			unicode.IsSpace(char)) {
			return false
		}
	}
	return true
}

// ValidatePasswordInput validates a strong password input for new accounts,
// including a min length, at least 1 number and at least
// 1 special character.
func ValidatePasswordInput(input string) error {
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
		case !(unicode.IsSpace(char) ||
			unicode.IsLetter(char) ||
			unicode.IsNumber(char) ||
			unicode.IsPunct(char) ||
			unicode.IsSymbol(char)):
			return errors.New("password must only contain alphanumeric characters, punctuation, or symbols")
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
