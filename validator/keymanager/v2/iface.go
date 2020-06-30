package v2

import (
	"context"
	"io"
)

// IKeymanager defines a general keymanager-v2 interface for Prysm wallets.
type IKeymanager interface {
	CreateAccount(ctx context.Context, password string) error
	ConfigFile(ctx context.Context) ([]byte, error)
}

// InitializeFromConfig reads a keymanager configuration file and
// initializes the appropriate keymanager from its contents.
func InitializeFromConfig(r io.Reader) (IKeymanager, error) {
	return nil, nil
}
