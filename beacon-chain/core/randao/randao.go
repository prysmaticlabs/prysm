// Package randao contains libraries to update and proposer's RANDAO layer
// and mixes the RANDAO with the existing RANDAO value in state.
package randao

import (
	"fmt"

	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// UpdateRandaoLayers increments the randao layer of the block proposer at the given slot.
func UpdateRandaoLayers(state *pb.BeaconState, slot uint64) (*pb.BeaconState, error) {
	vreg := state.ValidatorRegistry
	proposerIndex, err := v.BeaconProposerIdx(state, slot)
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
	latestMixesLength := params.BeaconConfig().LatestRandaoMixesLength
	prevMixes := state.LatestRandaoMixesHash32S[(state.Slot-1)%latestMixesLength]
	state.LatestRandaoMixesHash32S[state.Slot%latestMixesLength] = prevMixes
	return state
}
