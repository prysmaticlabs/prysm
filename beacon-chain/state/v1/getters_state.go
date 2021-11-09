package v1

import (
	"github.com/pkg/errors"
	customtypes "github.com/prysmaticlabs/prysm/beacon-chain/state/custom-types"
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

	return &ethpb.BeaconState{
		Fork:                        b.fork(),
		LatestBlockHeader:           b.latestBlockHeader(),
		Eth1Data:                    b.eth1Data(),
		Eth1DataVotes:               b.eth1DataVotes(),
		Validators:                  b.validators(),
		PreviousEpochAttestations:   b.previousEpochAttestations(),
		CurrentEpochAttestations:    b.currentEpochAttestations(),
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint(),
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint(),
		FinalizedCheckpoint:         b.finalizedCheckpoint(),
	}
}

// hasInnerState detects if the internal reference to the state data structure
// is populated correctly. Returns false if nil.
func (b *BeaconState) hasInnerState() bool {
	return b != nil && b.state != nil
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() *[8192][32]byte {
	if !b.hasInnerState() {
		return nil
	}
	if b.stateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	roots := [8192][32]byte(*b.stateRootsInternal())
	return &roots
}

// stateRootsInternal kept track of in the beacon state.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootsInternal() *customtypes.StateRoots {
	if !b.hasInnerState() {
		return nil
	}
	return b.stateRoots
}

// StateRootAtIndex retrieves a specific state root based on an
// input index value.
func (b *BeaconState) StateRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	if b.stateRoots == nil {
		return [32]byte{}, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.stateRootAtIndex(idx)
}

// stateRootAtIndex retrieves a specific state root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootAtIndex(idx uint64) ([32]byte, error) {
	if !b.hasInnerState() {
		return [32]byte{}, ErrNilInnerState
	}
	sRoots := make([][]byte, len(b.stateRoots))
	for i := range sRoots {
		sRoots[i] = b.stateRoots[i][:]
	}
	root, err := bytesutil.SafeCopyRootAtIndex(sRoots, idx)
	if err != nil {
		return [32]byte{}, err
	}
	return bytesutil.ToBytes32(root), nil
}

// MarshalSSZ marshals the underlying beacon state to bytes.
func (b *BeaconState) MarshalSSZ() ([]byte, error) {
	if !b.hasInnerState() {
		return nil, errors.New("nil beacon state")
	}
	return b.state.MarshalSSZ()
}

// ProtobufBeaconState transforms an input into beacon state in the form of protobuf.
// Error is returned if the input is not type protobuf beacon state.
func ProtobufBeaconState(s interface{}) (*ethpb.BeaconState, error) {
	pbState, ok := s.(*ethpb.BeaconState)
	if !ok {
		return nil, errors.New("input is not type ethpb.BeaconState")
	}
	return pbState, nil
}
