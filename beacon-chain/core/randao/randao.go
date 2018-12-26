package randao

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// UpdateRandaoLayers increments the randao layer of the block proposer at the given slot.
func UpdateRandaoLayers(state *pb.BeaconState, slot uint64) (*pb.BeaconState, error) {
	newState := proto.Clone(state).(*pb.BeaconState)
	vreg := newState.GetValidatorRegistry()

	proposerIndex, err := v.BeaconProposerIndex(newState, slot)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve proposer index %v", err)
	}
	vreg[proposerIndex].RandaoLayers++
	state.ValidatorRegistry = vreg
	return newState, nil
}

// UpdateRandaoMixes updates the latest randao mix hashes of the state.
func UpdateRandaoMixes(state *pb.BeaconState) *pb.BeaconState {
	slot := state.Slot
	randaoMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	state.LatestRandaoMixesHash32S[slot%randaoMixesLength] = state.LatestRandaoMixesHash32S[(slot-1)%randaoMixesLength]
	return state
}
