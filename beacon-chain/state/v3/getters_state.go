package v3

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// InnerStateUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) InnerStateUnsafe() interface{} {
	if b == nil {
		return nil
	}
	return b.state
}

// CloneInnerState the beacon state into a protobuf for usage.
func (b *BeaconState) CloneInnerState() interface{} {
	if b == nil || b.state == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()
	return &ethpb.BeaconStateBellatrix{
		GenesisTime:                  b.genesisTime(),
		GenesisValidatorsRoot:        b.genesisValidatorsRoot(),
		Slot:                         b.slot(),
		Fork:                         b.fork(),
		LatestBlockHeader:            b.latestBlockHeader(),
		BlockRoots:                   b.blockRoots(),
		StateRoots:                   b.stateRoots(),
		HistoricalRoots:              b.historicalRoots(),
		Eth1Data:                     b.eth1Data(),
		Eth1DataVotes:                b.eth1DataVotes(),
		Eth1DepositIndex:             b.eth1DepositIndex(),
		Validators:                   b.validators(),
		Balances:                     b.balances(),
		RandaoMixes:                  b.randaoMixes(),
		Slashings:                    b.slashings(),
		CurrentEpochParticipation:    b.currentEpochParticipation(),
		PreviousEpochParticipation:   b.previousEpochParticipation(),
		JustificationBits:            b.justificationBits(),
		PreviousJustifiedCheckpoint:  b.previousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:   b.currentJustifiedCheckpoint(),
		FinalizedCheckpoint:          b.finalizedCheckpoint(),
		InactivityScores:             b.inactivityScores(),
		CurrentSyncCommittee:         b.currentSyncCommittee(),
		NextSyncCommittee:            b.nextSyncCommittee(),
		LatestExecutionPayloadHeader: b.latestExecutionPayloadHeader(),
	}
}

// hasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) hasInnerState() bool {
	return b != nil && b.state != nil
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.StateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRoots()
}

// StateRoots kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRoots() [][]byte {
	if !b.hasInnerState() {
		return nil
	}
	return bytesutil.SafeCopy2dBytes(b.state.StateRoots)
}

// StateRootAtIndex retrieves a specific state root based on an
// input index value.
func (b *BeaconState) StateRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.StateRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRootAtIndex(idx)
}

// stateRootAtIndex retrieves a specific state root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootAtIndex(idx uint64) ([]byte, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	return bytesutil.SafeCopyRootAtIndex(b.state.StateRoots, idx)
}

// MarshalSSZ marshals the underlying beacon state to bytes.
func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, errors.New("nil beacon state")
	}
	return b.state.MarshalSSZ()
}

// ProtobufBeaconState transforms an input into beacon state Bellatrix in the form of protobuf.
// Error is returned if the input is not type protobuf beacon state.
func ProtobufBeaconState(s interface{}) (*ethpb.BeaconStateBellatrix, error) {
	pbState, ok := s.(*ethpb.BeaconStateBellatrix)
	if !ok {
		return nil, errors.New("input is not type pb.BeaconStateBellatrix")
	}
	return pbState, nil
}
