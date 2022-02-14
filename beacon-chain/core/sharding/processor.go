package sharding

import (
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
)

func getActiveShardCount(st state.BeaconState, epoch types.Epoch) uint64 {
	return 256
}

func appendIntermediateBlock(st state.BeaconState, blk block.BeaconBlock) {
	// Check state version
	if time.IsIntermediateBlockSlot(blk.Slot()) {
		// state.blocks_since_intermediate_block = []
	}
	// state.blocks_since_intermediate_block.append(block)
}

func processShardedData(st state.BeaconState, blk block.BeaconBlock) (state.BeaconState, error) {
	return nil, nil
}
