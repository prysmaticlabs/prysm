package v1

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/copyutil"

	"github.com/pkg/errors"
	pbp2p "github.com/prysmaticlabs/prysm/proto/proto/prysm/v2"
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
	return &pbp2p.BeaconState{
		GenesisTime:                 b.genesisTime(),
		GenesisValidatorsRoot:       b.genesisValidatorRoot(),
		Slot:                        b.slot(),
		Fork:                        b.fork(),
		LatestBlockHeader:           b.latestBlockHeader(),
		BlockRoots:                  b.blockRoots(),
		StateRoots:                  b.stateRoots(),
		HistoricalRoots:             b.historicalRoots(),
		Eth1Data:                    b.eth1Data(),
		Eth1DataVotes:               b.eth1DataVotes(),
		Eth1DepositIndex:            b.eth1DepositIndex(),
		Validators:                  b.validators(),
		Balances:                    b.balances(),
		RandaoMixes:                 b.randaoMixes(),
		Slashings:                   b.slashings(),
		PreviousEpochAttestations:   b.previousEpochAttestations(),
		CurrentEpochAttestations:    b.currentEpochAttestations(),
		JustificationBits:           b.justificationBits(),
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
	return b.safeCopy2DByteSlice(b.state.StateRoots)
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
	return b.safeCopyBytesAtIndex(b.state.StateRoots, idx)
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
func ProtobufBeaconState(s interface{}) (*pbp2p.BeaconState, error) {
	pbState, ok := s.(*pbp2p.BeaconState)
	if !ok {
		return nil, errors.New("input is not type pb.BeaconState")
	}
	return pbState, nil
}

func (b *BeaconState) safeCopy2DByteSlice(input [][]byte) [][]byte {
	if input == nil {
		return nil
	}

	dst := make([][]byte, len(input))
	for i, r := range input {
		tmp := make([]byte, len(r))
		copy(tmp, r)
		dst[i] = tmp
	}
	return dst
}

func (b *BeaconState) safeCopyBytesAtIndex(input [][]byte, idx uint64) ([]byte, error) {
	if input == nil {
		return nil, nil
	}

	if uint64(len(input)) <= idx {
		return nil, fmt.Errorf("index %d out of range", idx)
	}
	root := make([]byte, 32)
	copy(root, input[idx])
	return root, nil
}

func (b *BeaconState) safeCopyPendingAttestationSlice(input []*pbp2p.PendingAttestation) []*pbp2p.PendingAttestation {
	if input == nil {
		return nil
	}

	res := make([]*pbp2p.PendingAttestation, len(input))
	for i := 0; i < len(res); i++ {
		res[i] = copyutil.CopyPendingAttestation(input[i])
	}
	return res
}

func (b *BeaconState) safeCopyCheckpoint(input *ethpb.Checkpoint) *ethpb.Checkpoint {
	if input == nil {
		return nil
	}

	return copyutil.CopyCheckpoint(input)
}
