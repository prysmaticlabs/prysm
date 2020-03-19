package helpers

import (
	"fmt"
	"time"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// SlotToEpoch returns the epoch number of the input slot.
//
// Spec pseudocode definition:
//  def compute_epoch_of_slot(slot: Slot) -> Epoch:
//    """
//    Return the epoch number of ``slot``.
//    """
//    return Epoch(slot // SLOTS_PER_EPOCH)
func SlotToEpoch(slot uint64) uint64 {
	return slot / params.BeaconConfig().SlotsPerEpoch
}

// CurrentEpoch returns the current epoch number calculated from
// the slot number stored in beacon state.
//
// Spec pseudocode definition:
//  def get_current_epoch(state: BeaconState) -> Epoch:
//    """
//    Return the current epoch.
//    """
//    return compute_epoch_of_slot(state.slot)
func CurrentEpoch(state *stateTrie.BeaconState) uint64 {
	return SlotToEpoch(state.Slot())
}

// PrevEpoch returns the previous epoch number calculated from
// the slot number stored in beacon state. It alswo checks for
// underflow condition.
//
// Spec pseudocode definition:
//  def get_previous_epoch(state: BeaconState) -> Epoch:
//    """`
//    Return the previous epoch (unless the current epoch is ``GENESIS_EPOCH``).
//    """
//    current_epoch = get_current_epoch(state)
//    return GENESIS_EPOCH if current_epoch == GENESIS_EPOCH else Epoch(current_epoch - 1)
func PrevEpoch(state *stateTrie.BeaconState) uint64 {
	currentEpoch := CurrentEpoch(state)
	if currentEpoch == 0 {
		return 0
	}
	return currentEpoch - 1
}

// NextEpoch returns the next epoch number calculated form
// the slot number stored in beacon state.
func NextEpoch(state *stateTrie.BeaconState) uint64 {
	return SlotToEpoch(state.Slot()) + 1
}

// StartSlot returns the first slot number of the
// current epoch.
//
// Spec pseudocode definition:
//  def compute_start_slot_of_epoch(epoch: Epoch) -> Slot:
//    """
//    Return the start slot of ``epoch``.
//    """
//    return Slot(epoch * SLOTS_PER_EPOCH
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

// SlotsSinceEpochStarts returns number of slots since the start of the epoch.
func SlotsSinceEpochStarts(slot uint64) uint64 {
	return slot - StartSlot(SlotToEpoch(slot))
}

// Allow for slots "from the future" within a certain tolerance.
const timeShiftTolerance = 10 // ms

// VerifySlotTime validates the input slot is not from the future.
func VerifySlotTime(genesisTime uint64, slot uint64) error {
	slotTime := genesisTime + slot*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(roughtime.Now().Unix())
	if slotTime > currentTime+timeShiftTolerance {
		return fmt.Errorf("could not process slot from the future, slot time %d > current time %d", slotTime, currentTime)
	}
	return nil
}

// SlotsSince computes the number of time slots that have occurred since the given timestamp.
func SlotsSince(time time.Time) uint64 {
	return uint64(roughtime.Since(time).Seconds()) / params.BeaconConfig().SecondsPerSlot
}

// RoundUpToNearestEpoch rounds up the provided slot value to the nearest epoch.
func RoundUpToNearestEpoch(slot uint64) uint64 {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 {
		slot -= slot % params.BeaconConfig().SlotsPerEpoch
		slot += params.BeaconConfig().SlotsPerEpoch
	}
	return slot
}
