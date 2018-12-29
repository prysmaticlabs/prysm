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
	vreg := state.GetValidatorRegistry()

	proposerIndex, err := v.BeaconProposerIndex(state, slot)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve proposer index %v", err)
	}
	vreg[proposerIndex].RandaoLayers++
	state.ValidatorRegistry = vreg
	return state, nil
}

// UpdateRandaoMixes sets the beacon state's latest randao mixes according to the latest
// beacon slot.
func UpdateRandaoMixes(state *pb.BeaconState) *pb.BeaconState {
	newState := proto.Clone(state).(*pb.BeaconState)
	latestMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	prevMixes := state.LatestRandaoMixesHash32S[(newState.GetSlot()-1)%latestMixesLength]
	newState.LatestRandaoMixesHash32S[state.GetSlot()%latestMixesLength] = prevMixes
	return newState
}
