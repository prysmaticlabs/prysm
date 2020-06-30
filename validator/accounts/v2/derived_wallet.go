package v2

import (
	"errors"
	"io"
)

// CreateDerivedWallet -- unimplemented.
func CreateDerivedWallet(walletWriter io.Writer, passwordsWriter io.Writer, password string) error {
	return errors.New("unimplemented, work in progress")
}
