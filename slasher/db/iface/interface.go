package iface

import (
	"context"
	"io"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
)

// SlasherDB defines the necessary methods for a Prysm slasher DB.
type SlasherDB interface {
	io.Closer
	DatabasePath() string
	ClearDB() error

	// AttesterSlashing related methods.
	AttesterSlashings(ctx context.Context, status types.SlashingStatus) ([]*ethpb.AttesterSlashing, error)
	DeleteAttesterSlashing(ctx context.Context, attesterSlashing *ethpb.AttesterSlashing) error
	HasAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) (bool, types.SlashingStatus, error)
	SaveAttesterSlashing(ctx context.Context, status types.SlashingStatus, slashing *ethpb.AttesterSlashing) error
	SaveAttesterSlashings(ctx context.Context, status types.SlashingStatus, slashings []*ethpb.AttesterSlashing) error
	GetLatestEpochDetected(ctx context.Context) (uint64, error)
	SetLatestEpochDetected(ctx context.Context, epoch uint64) error

	// BlockHeader related methods.
	BlockHeaders(ctx context.Context, epoch uint64, validatorID uint64) ([]*ethpb.SignedBeaconBlockHeader, error)
	HasBlockHeader(ctx context.Context, epoch uint64, validatorID uint64) bool
	SaveBlockHeader(ctx context.Context, epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error
	DeleteBlockHeader(ctx context.Context, epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error

	// IndexedAttestations related methods.
	IdxAttsForTargetFromID(ctx context.Context, targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error)
	IdxAttsForTarget(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error)
	LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error)
	LatestValidatorIdx(ctx context.Context) (uint64, error)
	DoubleVotes(ctx context.Context, validatorIdx uint64, dataRoot []byte, origAtt *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
	HasIndexedAttestation(ctx context.Context, targetEpoch uint64, validatorID uint64) (bool, error)
	SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error
	DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error

	// MinMaxSpan related methods.
	ValidatorSpansMap(ctx context.Context, validatorIdx uint64) (*slashpb.EpochSpanMap, error)
	SaveValidatorSpansMap(ctx context.Context, validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error
	SaveCachedSpansMaps(ctx context.Context) error
	DeleteValidatorSpanMap(ctx context.Context, validatorIdx uint64) error

	// ProposerSlashing related methods.
	ProposalSlashingsByStatus(ctx context.Context, status types.SlashingStatus) ([]*ethpb.ProposerSlashing, error)
	DeleteProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) error
	HasProposerSlashing(ctx context.Context, slashing *ethpb.ProposerSlashing) (bool, types.SlashingStatus, error)
	SaveProposerSlashing(ctx context.Context, status types.SlashingStatus, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashings(ctx context.Context, status types.SlashingStatus, slashings []*ethpb.ProposerSlashing) error

	// Validator Index -> Pubkey related methods.
	ValidatorPubKey(ctx context.Context, validatorID uint64) ([]byte, error)
	SavePubKey(ctx context.Context, validatorID uint64, pubKey []byte) error
	DeletePubKey(ctx context.Context, validatorID uint64) error
}
