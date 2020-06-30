package v2

import (
	"errors"
	"io"
)

// CreateRemoteWallet -- unimplemented.
func CreateRemoteWallet(walletWriter io.Writer, passwordsWriter io.Writer, password string) error {
	return errors.New("unimplemented, work in progress")
}
