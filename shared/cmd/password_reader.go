package cmd

import (
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

// PasswordReader reads a password from a mock or stdin.
type PasswordReader interface {
	ReadPassword() (string, error)
}

// StdInPasswordReader reads a password from stdin.
type StdInPasswordReader struct {
}

// ReadPassword reads a password from stdin.
func (pr StdInPasswordReader) ReadPassword() (string, error) {
	pwd, error := terminal.ReadPassword(int(os.Stdin.Fd()))
	return string(pwd), error
}
