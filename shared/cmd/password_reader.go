package cmd

import (
	"golang.org/x/crypto/ssh/terminal"
	"os"
)

type PasswordReader interface {
	ReadPassword() (string, error)
}

type StdInPasswordReader struct {
}

func (pr StdInPasswordReader) ReadPassword() (string, error) {
	pwd, error := terminal.ReadPassword(int(os.Stdin.Fd()))
	return string(pwd), error
}
