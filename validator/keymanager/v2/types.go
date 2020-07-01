package v2

import (
	"context"
)

// IKeymanager defines a general keymanager-v2 interface for Prysm wallets.
type IKeymanager interface {
	CreateAccount(ctx context.Context, password string) error
	ConfigFile(ctx context.Context) ([]byte, error)
}

// Kind defines an enum for either direct, derived, or remote-signing
// keystores for Prysm wallets.
type Kind int

const (
	// Direct keymanager defines an on-disk, encrypted keystore-capable store.
	Direct Kind = iota
	// Derived keymanager defines a hierarchical-deterministic store.
	Derived
	// Remote keymanager capable of remote-signing data.
	Remote
)

// String marshals a keymanager kind to a string value.
func (k Kind) String() string {
	switch k {
	case Direct:
		return "direct"
	case Derived:
		return "derived"
	case Remote:
		return "remote"
	default:
		return ""
	}
}
