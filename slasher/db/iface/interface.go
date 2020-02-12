package iface

import (
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
	AttesterSlashings(status types.SlashingStatus) ([]*ethpb.AttesterSlashing, error)
	DeleteAttesterSlashing(attesterSlashing *ethpb.AttesterSlashing) error
	HasAttesterSlashing(slashing *ethpb.AttesterSlashing) (bool, types.SlashingStatus, error)
	SaveAttesterSlashing(status types.SlashingStatus, slashing *ethpb.AttesterSlashing) error
	SaveAttesterSlashings(status types.SlashingStatus, slashings []*ethpb.AttesterSlashing) error
	GetLatestEpochDetected() (uint64, error)
	SetLatestEpochDetected(epoch uint64) error

	// BlockHeader related methods.
	BlockHeaders(epoch uint64, validatorID uint64) ([]*ethpb.SignedBeaconBlockHeader, error)
	HasBlockHeader(epoch uint64, validatorID uint64) bool
	SaveBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error
	DeleteBlockHeader(epoch uint64, validatorID uint64, blockHeader *ethpb.SignedBeaconBlockHeader) error

	// IndexedAttestations related methods.
	IdxAttsForTargetFromID(targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error)
	IdxAttsForTarget(targetEpoch uint64) ([]*ethpb.IndexedAttestation, error)
	LatestIndexedAttestationsTargetEpoch() (uint64, error)
	LatestValidatorIdx() (uint64, error)
	DoubleVotes(validatorIdx uint64, dataRoot []byte, origAtt *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error)
	HasIndexedAttestation(targetEpoch uint64, validatorID uint64) (bool, error)
	SaveIndexedAttestation(idxAttestation *ethpb.IndexedAttestation) error
	DeleteIndexedAttestation(idxAttestation *ethpb.IndexedAttestation) error

	// MinMaxSpan related methods.
	ValidatorSpansMap(validatorIdx uint64) (*slashpb.EpochSpanMap, error)
	SaveValidatorSpansMap(validatorIdx uint64, spanMap *slashpb.EpochSpanMap) error
	SaveCachedSpansMaps() error
	DeleteValidatorSpanMap(validatorIdx uint64) error

	// ProposerSlashing related methods.
	ProposalSlashingsByStatus(status types.SlashingStatus) ([]*ethpb.ProposerSlashing, error)
	DeleteProposerSlashing(slashing *ethpb.ProposerSlashing) error
	HasProposerSlashing(slashing *ethpb.ProposerSlashing) (bool, types.SlashingStatus, error)
	SaveProposerSlashing(status types.SlashingStatus, slashing *ethpb.ProposerSlashing) error
	SaveProposerSlashings(status types.SlashingStatus, slashings []*ethpb.ProposerSlashing) error

	// Validator Index -> Pubkey related methods.
	ValidatorPubKey(validatorID uint64) ([]byte, error)
	SavePubKey(validatorID uint64, pubKey []byte) error
	DeletePubKey(validatorID uint64) error
}
