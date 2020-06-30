package v2

import (
	"context"

	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

// WalletConfig --
type WalletConfig struct {
	PasswordsDir string
	WalletDir    string
	Keymanager   v2keymanager.IKeymanager
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) error {
	return nil
}
