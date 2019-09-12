package helpers

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// BlockRootAtSlot returns the block root stored in the BeaconState for a recent slot.
// It returns an error if the requested block root is not within the slot range.
//
// Spec pseudocode definition:
//  def get_block_root_at_slot(state: BeaconState, slot: Slot) -> Hash:
//    """
//    Return the block root at a recent ``slot``.
//    """
//    assert slot < state.slot <= slot + SLOTS_PER_HISTORICAL_ROOT
//    return state.block_roots[slot % SLOTS_PER_HISTORICAL_ROOT]
func BlockRootAtSlot(state *pb.BeaconState, slot uint64) ([]byte, error) {
	if slot >= state.Slot || state.Slot > slot+params.BeaconConfig().SlotsPerHistoricalRoot {
		return []byte{}, errors.New("slot out of bounds")
	}
	return state.BlockRoots[slot%params.BeaconConfig().SlotsPerHistoricalRoot], nil
}

// BlockRoot returns the block root stored in the BeaconState for epoch start slot.
//
// Spec pseudocode definition:
//  def get_block_root(state: BeaconState, epoch: Epoch) -> Hash:
//    """
//    Return the block root at the start of a recent ``epoch``.
//    """
//    return get_block_root_at_slot(state, compute_start_slot_of_epoch(epoch))
func BlockRoot(state *pb.BeaconState, epoch uint64) ([]byte, error) {
	return BlockRootAtSlot(state, StartSlot(epoch))
}
