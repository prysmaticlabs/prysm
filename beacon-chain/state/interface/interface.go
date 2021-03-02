// Package iface defines the actual beacon state interface used
// by a Prysm beacon node, also containing useful, scoped interfaces such as
// a ReadOnlyState.
package iface

import (
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// ReadOnlyValidator defines a struct which only has read access to validator methods.
type ReadOnlyValidator interface {
	EffectiveBalance() uint64
	ActivationEligibilityEpoch() types.Epoch
	ActivationEpoch() types.Epoch
	WithdrawableEpoch() types.Epoch
	ExitEpoch() types.Epoch
	PublicKey() [48]byte
	WithdrawalCredentials() []byte
	Slashed() bool
	CopyValidator() *ethpb.Validator
	IsNil() bool
}

// ReadOnlyBeaconState defines a struct which only has read access to beacon state methods.
type ReadOnlyBeaconState interface {
	InnerStateUnsafe() *pbp2p.BeaconState
	CloneInnerState() *pbp2p.BeaconState
	HasInnerState() bool
	GenesisTime() uint64
	GenesisValidatorRoot() []byte
	GenesisUnixTime() time.Time
	Slot() types.Slot
	Fork() *pbp2p.Fork
	LatestBlockHeader() *ethpb.BeaconBlockHeader
	ParentRoot() [32]byte
	BlockRoots() [][]byte
	BlockRootAtIndex(idx uint64) ([]byte, error)
	StateRoots() [][]byte
	StateRootAtIndex(idx uint64) ([]byte, error)
	HistoricalRoots() [][]byte
	Eth1Data() *ethpb.Eth1Data
	Eth1DataVotes() []*ethpb.Eth1Data
	Eth1DepositIndex() uint64
	Validators() []*ethpb.Validator
	ValidatorAtIndex(idx types.ValidatorIndex) (*ethpb.Validator, error)
	ValidatorAtIndexReadOnly(idx types.ValidatorIndex) (ReadOnlyValidator, error)
	ValidatorIndexByPubkey(key [48]byte) (types.ValidatorIndex, bool)
	PubkeyAtIndex(idx types.ValidatorIndex) [48]byte
	NumValidators() int
	ReadFromEveryValidator(f func(idx int, val ReadOnlyValidator) error) error
	Balances() []uint64
	BalanceAtIndex(idx types.ValidatorIndex) (uint64, error)
	BalancesLength()
	RandaoMixes() [][]byte
	RandaoMixAtIndex(idx uint64) ([]byte, error)
	RandaoMixesLength() int
	Slashings() []uint64
	PreviousEpochAttestations() []*pbp2p.PendingAttestation
	CurrentEpochAttestations() []*pbp2p.PendingAttestation
	JustificationBits() bitfield.Bitvector4
	PreviousJustifiedCheckpoint() *ethpb.Checkpoint
	CurrentJustifiedCheckpoint() *ethpb.Checkpoint
	MatchCurrentJustifiedCheckpoint(c *ethpb.Checkpoint) bool
	MatchPreviousJustifiedCheckpoint(c *ethpb.Checkpoint) bool
	FinalizedCheckpoint() *ethpb.Checkpoint
	FinalizedCheckpointEpoch() types.Epoch
}

// WriteOnlyBeaconState defines a struct which only has write access to beacon state methods.
type WriteOnlyBeaconState interface {
	SetGenesisTime(val uint64) error
	SetGenesisValidatorRoot(val []byte) error
	SetSlot(val types.Slot) error
	SetFork(val *pbp2p.Fork) error
	SetLatestBlockHeader(val *ethpb.BeaconBlockHeader) error
	SetBlockRoots(val [][]byte) error
	UpdateBlockRootAtIndex(idx uint64, blockRoot [32]byte) error
	SetStateRoots(val [][]byte) error
	UpdateStateRootAtIndex(idx uint64, stateRoot [32]byte) error
	SetHistoricalRoots(val [][]byte) error
	SetEth1Data(val *ethpb.Eth1Data) error
	SetEth1DataVotes(val []*ethpb.Eth1Data) error
	AppendEth1DataVotes(val *ethpb.Eth1Data) error
	SetEth1DepositIndex(val uint64) error
	SetValidators(val []*ethpb.Validator) error
	ApplyToEveryValidator(f func(idx int, val *ethpb.Validator) (bool, *ethpb.Validator, error)) error
	UpdateValidatorAtIndex(idx types.ValidatorIndex, val *ethpb.Validator) error
	SetValidatorIndexByPubkey(pubKey [48]byte, validatorIndex types.ValidatorIndex)
	SetBalances(val []uint64) error
	UpdateBalancesAtIndex(idx types.ValidatorIndex, val uint64) error
	SetRandaoMixes(val [][]byte) error
	UpdateRandaoMixesAtIndex(idx uint64, val []byte) error
	SetSlashings(val []uint64) error
	UpdateSlashingsAtIndex(idx, val uint64) error
	SetPreviousEpochAttestations(val []*pbp2p.PendingAttestation) error
	SetCurrentEpochAttestations(val []*pbp2p.PendingAttestation) error
	AppendHistoricalRoots(root [32]byte) error
	AppendCurrentEpochAttestations(val *pbp2p.PendingAttestation) error
	AppendPreviousEpochAttestations(val *pbp2p.PendingAttestation) error
	AppendValidator(val *ethpb.Validator) error
	AppendBalance(bal uint64) error
	SetJustificationBits(val bitfield.Bitvector4) error
	SetPreviousJustifiedCheckpoint(val *ethpb.Checkpoint) error
	SetCurrentJustifiedCheckpoint(val *ethpb.Checkpoint) error
	SetFinalizedCheckpoint(val *ethpb.Checkpoint) error
}
