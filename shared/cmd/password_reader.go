package cmd

import (
	"errors"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

// ErrNonInteractiveTerminal occurs when the StdInPasswordReader is used in a non-interactive
// environment.
var ErrNonInteractiveTerminal = errors.New("terminal is input is non-interactive")

// PasswordReader reads a password from a mock or stdin.
type PasswordReader interface {
	ReadPassword() (string, error)
}

// StdInPasswordReader reads a password from stdin.
type StdInPasswordReader struct {
}

// ReadPassword reads a password from stdin.
func (pr StdInPasswordReader) ReadPassword() (string, error) {
	if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		return "", ErrNonInteractiveTerminal
	}
	pwd, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	return string(pwd), err
}
