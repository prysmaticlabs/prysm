package randao

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/types"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
)

// UpdateRandaoLayers increments the randao of the block proposer at the given slot.
func UpdateRandaoLayers(state *types.BeaconState, slot uint64) (*types.BeaconState, error) {
	vreg := state.ValidatorRegistry()

	proposerIndex, err := v.BeaconProposerIndex(state.Proto(), slot)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve proposer index %v", err)
	}

	vreg[proposerIndex].RandaoLayers++
	state.SetValidatorRegistry(vreg)

	return state, nil
}
