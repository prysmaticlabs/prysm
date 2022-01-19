package v1

import (
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ToProtoUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) ToProtoUnsafe() interface{} {
	if b == nil {
		return nil
	}

	var bRoots [][]byte
	if b.blockRoots != nil {
		bRoots = make([][]byte, len(b.blockRoots))
		for i, r := range b.blockRoots {
			tmp := r
			bRoots[i] = tmp[:]
		}
	}
	var sRoots [][]byte
	if b.stateRoots != nil {
		sRoots = make([][]byte, len(b.stateRoots))
		for i, r := range b.stateRoots {
			tmp := r
			sRoots[i] = tmp[:]
		}
	}
	var hRoots [][]byte
	if b.historicalRoots != nil {
		hRoots = make([][]byte, len(b.historicalRoots))
		for i, r := range b.historicalRoots {
			tmp := r
			hRoots[i] = tmp[:]
		}
	}
	var mixes [][]byte
	if b.randaoMixes != nil {
		mixes = make([][]byte, len(b.randaoMixes))
		for i, m := range b.randaoMixes {
			tmp := m
			mixes[i] = tmp[:]
		}
	}

	return &ethpb.BeaconState{
		GenesisTime:                 b.genesisTime,
		GenesisValidatorsRoot:       b.genesisValidatorsRoot[:],
		Slot:                        b.slot,
		Fork:                        b.fork,
		LatestBlockHeader:           b.latestBlockHeader,
		BlockRoots:                  bRoots,
		StateRoots:                  sRoots,
		HistoricalRoots:             hRoots,
		Eth1Data:                    b.eth1Data,
		Eth1DataVotes:               b.eth1DataVotes,
		Eth1DepositIndex:            b.eth1DepositIndex,
		Validators:                  b.validators,
		Balances:                    b.balances,
		RandaoMixes:                 mixes,
		Slashings:                   b.slashings,
		PreviousEpochAttestations:   b.previousEpochAttestations,
		CurrentEpochAttestations:    b.currentEpochAttestations,
		JustificationBits:           b.justificationBits,
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpoint,
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpoint,
		FinalizedCheckpoint:         b.finalizedCheckpoint,
	}
}

// ToProto the beacon state into a protobuf for usage.
func (b *BeaconState) ToProto() interface{} {
	if b == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	gvr := b.genesisValidatorsRoot
	var bRoots [][]byte
	if b.blockRoots != nil {
		bRoots = make([][]byte, len(b.blockRoots))
		for i, r := range b.blockRoots {
			tmp := r
			bRoots[i] = tmp[:]
		}
	}
	var sRoots [][]byte
	if b.stateRoots != nil {
		sRoots = make([][]byte, len(b.stateRoots))
		for i, r := range b.stateRoots {
			tmp := r
			sRoots[i] = tmp[:]
		}
	}
	var hRoots [][]byte
	if b.historicalRoots != nil {
		hRoots = make([][]byte, len(b.historicalRootsVal()))
		for i, r := range b.historicalRootsVal() {
			tmp := r
			hRoots[i] = tmp[:]
		}
	}
	var mixes [][]byte
	if b.randaoMixes != nil {
		mixes = make([][]byte, len(b.randaoMixes))
		for i, m := range b.randaoMixes {
			tmp := m
			mixes[i] = tmp[:]
		}
	}

	return &ethpb.BeaconState{
		GenesisTime:                 b.genesisTime,
		GenesisValidatorsRoot:       gvr[:],
		Slot:                        b.slot,
		Fork:                        b.forkVal(),
		LatestBlockHeader:           b.latestBlockHeaderVal(),
		BlockRoots:                  bRoots,
		StateRoots:                  sRoots,
		HistoricalRoots:             hRoots,
		Eth1Data:                    b.eth1DataVal(),
		Eth1DataVotes:               b.eth1DataVotesVal(),
		Eth1DepositIndex:            b.eth1DepositIndex,
		Validators:                  b.validatorsVal(),
		Balances:                    b.balancesVal(),
		RandaoMixes:                 mixes,
		Slashings:                   b.slashingsVal(),
		PreviousEpochAttestations:   b.previousEpochAttestationsVal(),
		CurrentEpochAttestations:    b.currentEpochAttestationsVal(),
		JustificationBits:           b.justificationBitsVal(),
		PreviousJustifiedCheckpoint: b.previousJustifiedCheckpointVal(),
		CurrentJustifiedCheckpoint:  b.currentJustifiedCheckpointVal(),
		FinalizedCheckpoint:         b.finalizedCheckpointVal(),
	}
}

// StateRoots kept track of in the beacon state.
func (b *BeaconState) StateRoots() [][]byte {
	if b.stateRoots == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	roots := make([][]byte, len(b.stateRoots))
	for i, r := range b.stateRoots {
		tmp := r
		roots[i] = tmp[:]
	}

	return roots
}

// StateRootAtIndex retrieves a specific state root based on an
// input index value.
func (b *BeaconState) StateRootAtIndex(idx uint64) ([]byte, error) {
	if b.stateRoots == nil {
		return nil, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	r, err := b.stateRootAtIndex(idx)
	if err != nil {
		return nil, err
	}
	return r[:], nil
}

// stateRootAtIndex retrieves a specific state root based on an
// input index value.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) stateRootAtIndex(idx uint64) ([32]byte, error) {
	if uint64(len(b.stateRoots)) <= idx {
		return [32]byte{}, fmt.Errorf("index %d out of range", idx)
	}
	return b.stateRoots[idx], nil
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

// InnerStateUnsafe returns the pointer value of the underlying
// beacon state proto object, bypassing immutability. Use with care.
func (b *BeaconState) InnerStateUnsafe() interface{} {
	return b.ToProtoUnsafe()
}

// CloneInnerState the beacon state into a protobuf for usage.
func (b *BeaconState) CloneInnerState() interface{} {
	return b.ToProto()
}
