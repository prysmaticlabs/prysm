package v2

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

// IKeymanager defines a general keymanager-v2 interface for Prysm wallets.
type IKeymanager interface {
	// CreateAccount based on the keymanager's logic. Returns the account name.
	CreateAccount(ctx context.Context, password string) (string, error)
	// MarshalConfigFile for the keymanager's options.
	MarshalConfigFile(ctx context.Context) ([]byte, error)
	// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
	FetchValidatingPublicKeys() ([][48]byte, error)
	// Sign signs a message using a validator key.
	Sign(context.Context, interface{}) (bls.Signature, error)
}

// Kind defines an enum for either direct, derived, or remote-signing
// keystores for Prysm wallets.
type Kind int

const (
	// Direct keymanager defines an on-disk, encrypted keystore-capable store.
	Direct Kind = iota
	// Derived keymanager using a hierarchical-deterministic algorithm.
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
		return fmt.Sprintf("%d", int(k))
	}
}
