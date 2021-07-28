// Package iface defines an interface for the slasher database,
// providing more advanced interfaces such as a
// ReadOnlyDatabase.
package iface

import (
	"context"
	"io"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/backuputil"
	dbtypes "github.com/prysmaticlabs/prysm/slasher/db/types"
	slashertypes "github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
)

// ReadOnlyDatabase represents a read only database with functions that do not modify the DB.
type ReadOnlyDatabase interface {
	// AttesterSlashing related methods.
	AttesterSlashings(ctx context.Context, status dbtypes.SlashingStatus) ([]*ethpb.AttesterSlashing, error)
	DeleteAttesterSlashing(ctx context.Context, attesterSlashing *ethpb.AttesterSlashing) error
	HasAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) (bool, dbtypes.SlashingStatus, error)
	GetLatestEpochDetected(ctx context.Context) (types.Epoch, error)

	// BlockHeader related methods.
	BlockHeaders(ctx context.Context, slot types.Slot, validatorIndex types.ValidatorIndex) ([]*ethpb.SignedBeaconBlockHeader, error)
	HasBlockHeader(ctx context.Context, slot types.Slot, validatorIndex types.ValidatorIndex) bool

	// IndexedAttestations related methods.
	HasIndexedAttestation(ctx context.Context, att *ethpb.IndexedAttestation) (bool, error)
	IndexedAttestationsForTarget(ctx context.Context, targetEpoch types.Epoch) ([]*ethpb.IndexedAttestation, error)
	IndexedAttestationsWithPrefix(ctx context.Context, targetEpoch types.Epoch, sigBytes []byte) ([]*ethpb.IndexedAttestation, error)
	LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error)

	// Highest Attestation related methods.
	HighestAttestation(ctx context.Context, validatorID uint64) (*slashpb.HighestAttestation, error)

	// MinMaxSpan related methods.
	EpochSpans(ctx context.Context, epoch types.Epoch, fromCache bool) (*slashertypes.EpochStore, error)

	// ProposerSlashing related methods.
	ProposalSlashingsByStatus(ctx context.Context, status dbtypes.SlashingStatus) ([]*ethpb.ProposerSlashing, error)
	HasProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) (bool, dbtypes.SlashingStatus, error)

	// Validator Index -> Pubkey related methods.
	ValidatorPubKey(ctx context.Context, validatorIndex types.ValidatorIndex) ([]byte, error)

	// Chain data related methods.
	ChainHead(ctx context.Context) (*ethpb.ChainHead, error)

	// Cache management methods.
	RemoveOldestFromCache(ctx context.Context) uint64
}

// WriteAccessDatabase represents a write access database with only functions that can modify the DB.
type WriteAccessDatabase interface {
	// AttesterSlashing related methods.
	SaveAttesterSlashing(ctx context.Context, status dbtypes.SlashingStatus, slashing *ethpb.AttesterSlashing) error
	SaveAttesterSlashings(ctx context.Context, status dbtypes.SlashingStatus, slashings []*ethpb.AttesterSlashing) error
	SetLatestEpochDetected(ctx context.Context, epoch types.Epoch) error

	// BlockHeader related methods.
	SaveBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error
	DeleteBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error
	PruneBlockHistory(ctx context.Context, currentEpoch, pruningEpochAge types.Epoch) error

	// IndexedAttestations related methods.
	SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	SaveIndexedAttestations(ctx context.Context, idxAttestations []*ethpb.IndexedAttestation) error
	DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	PruneAttHistory(ctx context.Context, currentEpoch, pruningEpochAge types.Epoch) error

	// Highest Attestation related methods.
	SaveHighestAttestation(ctx context.Context, highest *slashpb.HighestAttestation) error

	// MinMaxSpan related methods.
	SaveEpochSpans(ctx context.Context, epoch types.Epoch, spans *slashertypes.EpochStore, toCache bool) error

	// ProposerSlashing related methods.
	DeleteProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashing(ctx context.Context, status dbtypes.SlashingStatus, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashings(ctx context.Context, status dbtypes.SlashingStatus, slashings []*ethpb.ProposerSlashing) error

	// Validator Index -> Pubkey related methods.
	SavePubKey(ctx context.Context, validatorIndex types.ValidatorIndex, pubKey []byte) error
	DeletePubKey(ctx context.Context, validatorIndex types.ValidatorIndex) error

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
	backuputil.BackupExporter
	FullAccessDatabase
	DatabasePath() string
	ClearDB() error
}

// EpochSpansStore represents a data access layer for marshaling and unmarshaling validator spans for each validator per epoch.
type EpochSpansStore interface {
	SetValidatorSpan(ctx context.Context, idx types.ValidatorIndex, newSpan slashertypes.Span) error
	GetValidatorSpan(ctx context.Context, idx types.ValidatorIndex) (slashertypes.Span, error)
}
