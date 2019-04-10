package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// SlotToEpoch returns the epoch number of the input slot.
//
// Spec pseudocode definition:
//   def slot_to_epoch(slot: SlotNumber) -> Epoch:
//    return slot // SLOTS_PER_EPOCH
func SlotToEpoch(slot uint64) uint64 {
	return slot / params.BeaconConfig().SlotsPerEpoch
}

// CurrentEpoch returns the current epoch number calculated from
// the slot number stored in beacon state.
//
// Spec pseudocode definition:
//   def get_current_epoch(state: BeaconState) -> Epoch:
//    return slot_to_epoch(state.slot)
func CurrentEpoch(state *pb.BeaconState) uint64 {
	return SlotToEpoch(state.Slot)
}

// PrevEpoch returns the previous epoch number calculated from
// the slot number stored in beacon state. It also checks for
// underflow condition.
//
// def get_previous_epoch(state: BeaconState) -> Epoch:
//    """`
//    Return the previous epoch of the given ``state``.
//    """
//    return max(get_current_epoch(state) - 1, GENESIS_EPOCH)
func PrevEpoch(state *pb.BeaconState) uint64 {
	if CurrentEpoch(state) > params.BeaconConfig().GenesisEpoch {
		return CurrentEpoch(state) - 1
	}
	return params.BeaconConfig().GenesisEpoch
}

// NextEpoch returns the next epoch number calculated form
// the slot number stored in beacon state.
func NextEpoch(state *pb.BeaconState) uint64 {
	return SlotToEpoch(state.Slot) + 1
}

// StartSlot returns the first slot number of the
// current epoch.
//
// Spec pseudocode definition:
//   def get_epoch_start_slot(epoch: Epoch) -> SlotNumber:
//    return epoch * SLOTS_PER_EPOCH
func StartSlot(epoch uint64) uint64 {
	return epoch * params.BeaconConfig().SlotsPerEpoch
}

// IsEpochStart returns true if the given slot number is an epoch starting slot
// number.
func IsEpochStart(slot uint64) bool {
	return slot%params.BeaconConfig().SlotsPerEpoch == 0
}

// IsEpochEnd returns true if the given slot number is an epoch ending slot
// number.
func IsEpochEnd(slot uint64) bool {
	return IsEpochStart(slot + 1)
}
