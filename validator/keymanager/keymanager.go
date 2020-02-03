package keymanager

import (
	"errors"

	"github.com/prysmaticlabs/prysm/shared/bls"
)

// ErrNoSuchKey is returned whenever a request is made for a key of which a key manager is unaware.
var ErrNoSuchKey = errors.New("no such key")

// ErrCannotSign is returned whenever a signing attempt fails.
var ErrCannotSign = errors.New("cannot sign")

// KeyManager controls access to private keys by the validator.
type KeyManager interface {
	// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
	FetchValidatingKeys() ([][48]byte, error)
	// Sign signs a message for the validator to broadcast.
	Sign(pubKey [48]byte, root [32]byte, domain uint64) (*bls.Signature, error)
}
