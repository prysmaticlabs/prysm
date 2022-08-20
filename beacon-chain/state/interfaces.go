// Package state defines the actual beacon state interface used
// by a Prysm beacon node, also containing useful, scoped interfaces such as
// a ReadOnlyState and WriteOnlyBeaconState.
package state

import (
	"context"

	"github.com/prysmaticlabs/go-bitfield"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// BeaconState has read and write access to beacon state methods.
type BeaconState interface {
	SpecParametersProvider
	ReadOnlyBeaconState
	WriteOnlyBeaconState
	Copy() BeaconState
	HashTreeRoot(ctx context.Context) ([32]byte, error)
	FutureForkStub
	StateProver
}

// SpecParametersProvider provides fork-specific configuration parameters as
// defined in the consensus specification for the beacon chain.
type SpecParametersProvider interface {
	InactivityPenaltyQuotient() (uint64, error)
	ProportionalSlashingMultiplier() (uint64, error)
}

// StateProver defines the ability to create Merkle proofs for beacon state fields.
type StateProver interface {
	FinalizedRootProof(ctx context.Context) ([][]byte, error)
	CurrentSyncCommitteeProof(ctx context.Context) ([][]byte, error)
	NextSyncCommitteeProof(ctx context.Context) ([][]byte, error)
}

// ReadOnlyBeaconState defines a struct which only has read access to beacon state methods.
type ReadOnlyBeaconState interface {
	ReadOnlyBlockRoots
	ReadOnlyStateRoots
	ReadOnlyRandaoMixes
	ReadOnlyEth1Data
	ReadOnlyValidators
	ReadOnlyBalances
	ReadOnlyCheckpoint
	ReadOnlyAttestations
	InnerStateUnsafe() interface{}
	CloneInnerState() interface{}
	GenesisTime() uint64
	GenesisValidatorsRoot() []byte
	Slot() types.Slot
	Fork() *ethpb.Fork
	LatestBlockHeader() *ethpb.BeaconBlockHeader
	HistoricalRoots() [][]byte
	Slashings() []uint64
	FieldReferencesCount() map[string]uint64
	MarshalSSZ() ([]byte, error)
	IsNil() bool
	Version() int
	LatestExecutionPayloadHeader() (*enginev1.ExecutionPayloadHeader, error)
}

// WriteOnlyBeaconState defines a struct which only has write access to beacon state methods.
type WriteOnlyBeaconState interface {
	WriteOnlyBlockRoots
	WriteOnlyStateRoots
	WriteOnlyRandaoMixes
	WriteOnlyEth1Data
	WriteOnlyValidators
	WriteOnlyBalances
	WriteOnlyCheckpoint
	WriteOnlyAttestations
	SetGenesisTime(val uint64) error
	SetGenesisValidatorsRoot(val []byte) error
	SetSlot(val types.Slot) error
	SetFork(val *ethpb.Fork) error
	SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error
	SetHistoricalRoots(val [][]byte) error
	SetSlashings(val []uint64) error
	UpdateSlashingsAtIndex(idx, val uint64) error
	AppendHistoricalRoots(root [32]byte) error
	SetLatestExecutionPayloadHeader(payload interfaces.ExecutionData) error
}

// ReadOnlyValidator defines a struct which only has read access to validator methods.
type ReadOnlyValidator interface {
	EffectiveBalance() uint64
	ActivationEligibilityEpoch() types.Epoch
	ActivationEpoch() types.Epoch
	WithdrawableEpoch() types.Epoch
	ExitEpoch() types.Epoch
	PublicKey() [fieldparams.BLSPubkeyLength]byte
	WithdrawalCredentials() []byte
	Slashed() bool
	IsNil() bool
}

// ReadOnlyValidators defines a struct which only has read access to validators methods.
type ReadOnlyValidators interface {
	Validators() []*ethpb.Validator
	ValidatorAtIndex(idx types.ValidatorIndex) (*ethpb.Validator, error)
	ValidatorAtIndexReadOnly(idx types.ValidatorIndex) (ReadOnlyValidator, error)
	ValidatorIndexByPubkey(key [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool)
	PubkeyAtIndex(idx types.ValidatorIndex) [fieldparams.BLSPubkeyLength]byte
	NumValidators() int
	ReadFromEveryValidator(f func(idx int, val ReadOnlyValidator) error) error
}

// ReadOnlyBalances defines a struct which only has read access to balances methods.
type ReadOnlyBalances interface {
	Balances() []uint64
	BalanceAtIndex(idx types.ValidatorIndex) (uint64, error)
	BalancesLength() int
}

// ReadOnlyCheckpoint defines a struct which only has read access to checkpoint methods.
type ReadOnlyCheckpoint interface {
	PreviousJustifiedCheckpoint() *ethpb.Checkpoint
	CurrentJustifiedCheckpoint() *ethpb.Checkpoint
	MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool
	MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool
	FinalizedCheckpoint() *ethpb.Checkpoint
	FinalizedCheckpointEpoch() types.Epoch
	JustificationBits() bitfield.Bitvector4
}

// ReadOnlyBlockRoots defines a struct which only has read access to block roots methods.
type ReadOnlyBlockRoots interface {
	BlockRoots() [][]byte
	BlockRootAtIndex(idx uint64) ([]byte, error)
}

// ReadOnlyStateRoots defines a struct which only has read access to state roots methods.
type ReadOnlyStateRoots interface {
	StateRoots() [][]byte
	StateRootAtIndex(idx uint64) ([]byte, error)
}

// ReadOnlyRandaoMixes defines a struct which only has read access to randao mixes methods.
type ReadOnlyRandaoMixes interface {
	RandaoMixes() [][]byte
	RandaoMixAtIndex(idx uint64) ([]byte, error)
	RandaoMixesLength() int
}

// ReadOnlyEth1Data defines a struct which only has read access to eth1 data methods.
type ReadOnlyEth1Data interface {
	Eth1Data() *ethpb.Eth1Data
	Eth1DataVotes() []*ethpb.Eth1Data
	Eth1DepositIndex() uint64
}

// ReadOnlyAttestations defines a struct which only has read access to attestations methods.
type ReadOnlyAttestations interface {
	PreviousEpochAttestations() ([]*ethpb.PendingAttestation, error)
	CurrentEpochAttestations() ([]*ethpb.PendingAttestation, error)
}

// WriteOnlyBlockRoots defines a struct which only has write access to block roots methods.
type WriteOnlyBlockRoots interface {
	SetBlockRoots(val [][]byte) error
	UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error
}

// WriteOnlyStateRoots defines a struct which only has write access to state roots methods.
type WriteOnlyStateRoots interface {
	SetStateRoots(val [][]byte) error
	UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error
}

// WriteOnlyEth1Data defines a struct which only has write access to eth1 data methods.
type WriteOnlyEth1Data interface {
	SetEth1Data(val *ethpb.Eth1Data) error
	SetEth1DataVotes(val []*ethpb.Eth1Data) error
	AppendEth1DataVotes(val *ethpb.Eth1Data) error
	SetEth1DepositIndex(val uint64) error
}

// WriteOnlyValidators defines a struct which only has write access to validators methods.
type WriteOnlyValidators interface {
	SetValidators(val []*ethpb.Validator) error
	ApplyToEveryValidator(f func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error)) error
	UpdateValidatorAtIndex(idx types.ValidatorIndex, val *ethpb.Validator) error
	AppendValidator(val *ethpb.Validator) error
}

// WriteOnlyBalances defines a struct which only has write access to balances methods.
type WriteOnlyBalances interface {
	SetBalances(val []uint64) error
	UpdateBalancesAtIndex(idx types.ValidatorIndex, val uint64) error
	AppendBalance(bal uint64) error
}

// WriteOnlyRandaoMixes defines a struct which only has write access to randao mixes methods.
type WriteOnlyRandaoMixes interface {
	SetRandaoMixes(val [][]byte) error
	UpdateRandaoMixesAtIndex(idx uint64, val []byte) error
}

// WriteOnlyCheckpoint defines a struct which only has write access to check point methods.
type WriteOnlyCheckpoint interface {
	SetFinalizedCheckpoint(val *ethpb.Checkpoint) error
	SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) error
	SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) error
	SetJustificationBits(val bitfield.Bitvector4) error
}

// WriteOnlyAttestations defines a struct which only has write access to attestations methods.
type WriteOnlyAttestations interface {
	AppendCurrentEpochAttestations(val *ethpb.PendingAttestation) error
	AppendPreviousEpochAttestations(val *ethpb.PendingAttestation) error
	RotateAttestations() error
}

// FutureForkStub defines methods that are used for future forks. This is a low cost solution to enable
// various state casting of interface to work.
type FutureForkStub interface {
	AppendCurrentParticipationBits(val byte) error
	AppendPreviousParticipationBits(val byte) error
	AppendInactivityScore(s uint64) error
	CurrentEpochParticipation() ([]byte, error)
	PreviousEpochParticipation() ([]byte, error)
	UnrealizedCheckpointBalances() (uint64, uint64, uint64, error)
	InactivityScores() ([]uint64, error)
	SetInactivityScores(val []uint64) error
	CurrentSyncCommittee() (*ethpb.SyncCommittee, error)
	SetCurrentSyncCommittee(val *ethpb.SyncCommittee) error
	SetPreviousParticipationBits(val []byte) error
	SetCurrentParticipationBits(val []byte) error
	ModifyCurrentParticipationBits(func(val []byte) ([]byte, error)) error
	ModifyPreviousParticipationBits(func(val []byte) ([]byte, error)) error
	NextSyncCommittee() (*ethpb.SyncCommittee, error)
	SetNextSyncCommittee(val *ethpb.SyncCommittee) error
}
