package keymanager

import (
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	FetchValidatingKeys() ([][params.KEY_BYTES_LENGTH]byte, error)
	// Sign signs a message for the validator to broadcast.
	Sign(pubKey [params.KEY_BYTES_LENGTH]byte, root [params.ROOT_BYTES_LENGTH]byte, domain uint64) (*bls.Signature, error)
}

// ProtectingKeyManager provides access to a keymanager that protects its clients from slashing events.
type ProtectingKeyManager interface {
	// SignProposal signs a block proposal for the validator to broadcast.
	SignProposal(pubKey [params.KEY_BYTES_LENGTH]byte, domain uint64, data *ethpb.BeaconBlockHeader) (*bls.Signature, error)

	// SignAttestation signs an attestation for the validator to broadcast.
	SignAttestation(pubKey [params.KEY_BYTES_LENGTH]byte, domain uint64, data *ethpb.AttestationData) (*bls.Signature, error)
}
