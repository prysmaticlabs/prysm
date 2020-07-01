package v2

import (
	"context"
)

// IKeymanager defines a general keymanager-v2 interface for Prysm wallets.
type IKeymanager interface {
	CreateAccount(ctx context.Context, password string) error
	ConfigFile(ctx context.Context) ([]byte, error)
}
