package keymanager

import (
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
)

// ErrNoSuchKey is returned whenever a request is made for a key of which a key manager is unaware.
var ErrNoSuchKey = errors.New("no such key")

// ErrCannotSign is returned whenever a signing attempt fails.
var ErrCannotSign = errors.New("cannot sign")

// ErrCouldSlash is returned whenever a signing attempt is refused due to a potential slashing event.
var ErrCouldSlash = errors.New("could result in a slashing event")

// KeyManager controls access to private keys by the validator.
type KeyManager interface {
	// FetchValidatingKeys fetches the list of public keys that should be used to validate with.
	FetchValidatingKeys() ([][48]byte, error)
	// Sign signs a message for the validator to broadcast.
	Sign(pubKey [48]byte, root [32]byte, domain uint64) (*bls.Signature, error)
}

// ProtectingKeyManager provides access to a keymanager that protects its clients from slashing events.
type ProtectingKeyManager interface {
	// SignProposal signs a block proposal for the validator to broadcast.
	SignProposal(pubKey [48]byte, domain uint64, data *ethpb.BeaconBlockHeader) (*bls.Signature, error)

	// SignAttestation signs an attestation for the validator to broadcast.
	SignAttestation(pubKey [48]byte, domain uint64, data *ethpb.AttestationData) (*bls.Signature, error)
}
