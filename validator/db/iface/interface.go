// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	"github.com/prysmaticlabs/prysm/validator/db/kv"
)

// ValidatorDB defines the necessary methods for a Prysm validator DB.
type ValidatorDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error
	UpdatePublicKeysBuckets(publicKeys [][48]byte) error

	// Genesis information related methods.
	GenesisValidatorsRoot(ctx context.Context) ([]byte, error)
	SaveGenesisValidatorsRoot(ctx context.Context, genValRoot []byte) error

	// Proposer history methods.
	HighestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	LowestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	ProposalHistoryForSlot(ctx context.Context, publicKey [48]byte, slot uint64) ([32]byte, bool, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey [48]byte, slot uint64, signingRoot []byte) error
	ProposedPublicKeys(ctx context.Context) ([][48]byte, error)

	// Attester history methods.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]kv.EncHistoryData, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKeys map[[48]byte]kv.EncHistoryData) error
	SaveAttestationHistoryForPubKey(ctx context.Context, pubKey [48]byte, history kv.EncHistoryData) error
	AttestedPublicKeys(ctx context.Context) ([][48]byte, error)
}
