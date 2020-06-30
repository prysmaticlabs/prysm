package v2

import (
	"context"
	"errors"

	v2keymanager "github.com/prysmaticlabs/prysm/validator/keymanager/v2"
)

// WalletConfig --
type WalletConfig struct {
	PasswordsDir string
	WalletDir    string
	Keymanager   v2keymanager.IKeymanager
}

// Wallet --
type Wallet struct {
	keymanager v2keymanager.IKeymanager
}

// CreateWallet given a set of configuration options, will leverage
// a keymanager to create and write a new wallet to disk for a Prysm validator.
func CreateWallet(ctx context.Context, cfg *WalletConfig) (*Wallet, error) {
	return &Wallet{
		keymanager: cfg.Keymanager,
	}, nil
}

// ReadWallet --
func ReadWallet(ctx context.Context, walletPath string) (*Wallet, error) {
	return &Wallet{}, errors.New("unimplemented")
}
