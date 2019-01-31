package helpers

import (
	"github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var config = params.BeaconConfig()

// SlotToEpoch returns the epoch number of the input slot.
//
// Spec pseudocode definition:
//   def slot_to_epoch(slot: SlotNumber) -> EpochNumber:
//    return slot // EPOCH_LENGTH
func SlotToEpoch(slot uint64) uint64 {
	return slot / config.EpochLength
}

// CurrentEpoch returns the current epoch number calculated from
// the slot number stored in beacon state.
//
// Spec pseudocode definition:
//   def get_current_epoch(state: BeaconState) -> EpochNumber:
//    return slot_to_epoch(state.slot)
func CurrentEpoch(state *ethereum_beacon_p2p_v1.BeaconState) uint64 {
	return SlotToEpoch(state.Slot)
}

// StartSlot returns the first slot number of the
// current epoch.
//
// Spec pseudocode definition:
//   def get_epoch_start_slot(epoch: EpochNumber) -> SlotNumber:
//    return epoch * EPOCH_LENGTH
func StartSlot(epoch uint64) uint64 {
	return epoch * config.EpochLength
}
