// Package iface defines an interface for the slasher database,
// providing more advanced interfaces such as a
// ReadOnlyDatabase.
package iface

import (
	"context"
	"io"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
	detectionTypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// ReadOnlyDatabase represents a read only database with functions that do not modify the DB.
type ReadOnlyDatabase interface {
	// AttesterSlashing related methods.
	AttesterSlashings(ctx context.Context, status types.SlashingStatus) ([]*ethpb.AttesterSlashing, error)
	DeleteAttesterSlashing(ctx context.Context, attesterSlashing *ethpb.AttesterSlashing) error
	HasAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) (bool, types.SlashingStatus, error)
	GetLatestEpochDetected(ctx context.Context) (uint64, error)

	// BlockHeader related methods.
	BlockHeaders(ctx context.Context, epoch uint64, validatorID uint64) ([]*ethpb.SignedBeaconBlockHeader, error)
	HasBlockHeader(ctx context.Context, epoch uint64, validatorID uint64) bool

	// IndexedAttestations related methods.
	HasIndexedAttestation(ctx context.Context, att *ethpb.IndexedAttestation) (bool, error)
	IndexedAttestationsForTarget(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error)
	IndexedAttestationsWithPrefix(ctx context.Context, targetEpoch uint64, sigBytes []byte) ([]*ethpb.IndexedAttestation, error)
	LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error)

	// MinMaxSpan related methods.
	EpochSpansMap(ctx context.Context, epoch uint64) (map[uint64]detectionTypes.Span, bool, error)
	EpochSpanByValidatorIndex(ctx context.Context, validatorIdx uint64, epoch uint64) (detectionTypes.Span, error)
	EpochsSpanByValidatorsIndices(ctx context.Context, validatorIndices []uint64, maxEpoch uint64) (map[uint64]map[uint64]detectionTypes.Span, error)

	// ProposerSlashing related methods.
	ProposalSlashingsByStatus(ctx context.Context, status types.SlashingStatus) ([]*ethpb.ProposerSlashing, error)
	HasProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) (bool, types.SlashingStatus, error)

	// Validator Index -> Pubkey related methods.
	ValidatorPubKey(ctx context.Context, validatorID uint64) ([]byte, error)

	// Chain data related methods.
	ChainHead(ctx context.Context) (*ethpb.ChainHead, error)
}

// WriteAccessDatabase represents a write access database with only functions that can modify the DB.
type WriteAccessDatabase interface {
	// AttesterSlashing related methods.
	SaveAttesterSlashing(ctx context.Context, status types.SlashingStatus, slashing *ethpb.AttesterSlashing) error
	SaveAttesterSlashings(ctx context.Context, status types.SlashingStatus, slashings []*ethpb.AttesterSlashing) error
	SetLatestEpochDetected(ctx context.Context, epoch uint64) error

	// BlockHeader related methods.
	SaveBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error
	DeleteBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error
	PruneBlockHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error

	// IndexedAttestations related methods.
	SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	SaveIndexedAttestations(ctx context.Context, idxAttestations []*ethpb.IndexedAttestation) error
	DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	PruneAttHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error

	// MinMaxSpan related methods.
	SaveEpochSpansMap(ctx context.Context, epoch uint64, spanMap map[uint64]detectionTypes.Span) error
	SaveValidatorEpochSpan(ctx context.Context, validatorIdx uint64, epoch uint64, spans detectionTypes.Span) error
	SaveCachedSpansMaps(ctx context.Context) error
	SaveEpochsSpanByValidatorsIndices(ctx context.Context, epochsSpans map[uint64]map[uint64]detectionTypes.Span) error
	DeleteEpochSpans(ctx context.Context, validatorIdx uint64) error
	DeleteValidatorSpanByEpoch(ctx context.Context, validatorIdx uint64, epoch uint64) error

	// ProposerSlashing related methods.
	DeleteProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashing(ctx context.Context, status types.SlashingStatus, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashings(ctx context.Context, status types.SlashingStatus, slashings []*ethpb.ProposerSlashing) error

	// Validator Index -> Pubkey related methods.
	SavePubKey(ctx context.Context, validatorID uint64, pubKey []byte) error
	DeletePubKey(ctx context.Context, validatorID uint64) error

	// Chain data related methods.
	SaveChainHead(ctx context.Context, head *ethpb.ChainHead) error
}

// FullAccessDatabase represents a full access database with only DB interaction functions.
type FullAccessDatabase interface {
	ReadOnlyDatabase
	WriteAccessDatabase
}

// Database represents a full access database with the proper DB helper functions.
type Database interface {
	io.Closer
	FullAccessDatabase

	DatabasePath() string
	ClearDB() error
}
