package v1

import (
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// ErrNoSuchKey is returned whenever a request is made for a key of which a key manager is unaware.
var ErrNoSuchKey = errors.New("no such key")

// ErrCannotSign is returned whenever a signing attempt fails.
var ErrCannotSign = errors.New("cannot sign")

// ErrDenied is returned whenever a signing attempt is denied.
var ErrDenied = errors.New("signing attempt denied")

// KeyManager controls access to private keys by the validator.
type KeyManager interface {
	// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
	FetchValidatingKeys() ([][48]byte, error)
	// Sign signs a message for the validator to broadcast.
	// Note that the domain should already be part of the root, but it is passed along for security purposes.
	Sign(pubKey [48]byte, root [32]byte) (bls.Signature, error)
}

// ProtectingKeyManager provides access to a keymanager that protects its clients from slashing events.
type ProtectingKeyManager interface {
	// SignGeneric signs a generic root.
	// Note that the domain should already be part of the root, but it is provided for authorisation purposes.
	SignGeneric(pubKey [48]byte, root [32]byte, domain [32]byte) (bls.Signature, error)

	// SignProposal signs a block proposal for the validator to broadcast.
	SignProposal(pubKey [48]byte, domain [32]byte, data *ethpb.BeaconBlockHeader) (bls.Signature, error)

	// SignAttestation signs an attestation for the validator to broadcast.
	SignAttestation(pubKey [48]byte, domain [32]byte, data *ethpb.AttestationData) (bls.Signature, error)
}
