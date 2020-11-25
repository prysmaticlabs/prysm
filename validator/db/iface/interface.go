// Package iface defines an interface for the validator database.
package iface

import (
	"context"
	"io"

	"github.com/prysmaticlabs/go-bitfield"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
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

	// Proposer protection related methods.
	ProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch uint64) (bitfield.Bitlist, error)
	SaveProposalHistoryForEpoch(ctx context.Context, publicKey []byte, epoch uint64, history bitfield.Bitlist) error

	// New data structure methods
	HighestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	LowestSignedProposal(ctx context.Context, publicKey [48]byte) (uint64, error)
	ProposalHistoryForSlot(ctx context.Context, publicKey [48]byte, slot uint64) ([]byte, error)
	SaveProposalHistoryForSlot(ctx context.Context, pubKey [48]byte, slot uint64, signingRoot []byte) error
	ProposedPublicKeys(ctx context.Context) ([][48]byte, error)

	// Attester protection related methods.
	AttestationHistoryForPubKeys(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]*slashpb.AttestationHistory, error)
	SaveAttestationHistoryForPubKeys(ctx context.Context, historyByPubKey map[[48]byte]*slashpb.AttestationHistory) error
	HighestSignedTargetEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	HighestSignedSourceEpoch(ctx context.Context, publicKey [48]byte) (uint64, error)
	SaveHighestSignedTargetEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) uint64
	SaveHighestSignedSourceEpoch(ctx context.Context, publicKey [48]byte, epoch uint64) uint64

	// New attestation store methods.
	AttestationHistoryForPubKeysV2(ctx context.Context, publicKeys [][48]byte) (map[[48]byte]kv.EncHistoryData, error)
	SaveAttestationHistoryForPubKeysV2(ctx context.Context, historyByPubKeys map[[48]byte]kv.EncHistoryData) error
	SaveAttestationHistoryForPubKeyV2(ctx context.Context, pubKey [48]byte, history kv.EncHistoryData) error
	AttestedPublicKeys(ctx context.Context) ([][48]byte, error)
}
