package cmd

import (
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

// Reads a password from a mock or stdin.
type PasswordReader interface {
	ReadPassword() (string, error)
}

// Reads a password from stdin.
type StdInPasswordReader struct {
}

func (pr StdInPasswordReader) ReadPassword() (string, error) {
	pwd, error := terminal.ReadPassword(int(os.Stdin.Fd()))
	return string(pwd), error
}
