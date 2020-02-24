package iface

import (
	"context"
	"io"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
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
	IndexedAttestations(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error)
	IdxAttsForTargetFromID(ctx context.Context, targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error)
	IdxAttsForTarget(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error)
	LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error)
	LatestValidatorIdx(ctx context.Context) (uint64, error)
	DoubleVotes(ctx context.Context, validatorIdx uint64, dataRoot []byte, origAtt *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
	HasIndexedAttestation(ctx context.Context, targetEpoch uint64, validatorID uint64) (bool, error)

	// MinMaxSpan related methods.
	ValidatorSpansMap(ctx context.Context, validatorIdx uint64) (*slashpb.EpochSpanMap, error)

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
	SaveBlockHeader(ctx context.Context, epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error
	DeleteBlockHeader(ctx context.Context, epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error
	PruneBlockHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error

	// IndexedAttestations related methods.
	SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	SaveIncomingIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	SaveIncomingIndexedAttestations(ctx context.Context, idxAttestations []*ethpb.IndexedAttestation) error
	DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	PruneAttHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error

	// MinMaxSpan related methods.
	SaveValidatorSpansMap(ctx context.Context, validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error
	SaveCachedSpansMaps(ctx context.Context) error
	DeleteValidatorSpanMap(ctx context.Context, validatorIdx uint64) error

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
