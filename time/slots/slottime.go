package slots

import (
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	mathutil "github.com/prysmaticlabs/prysm/v3/math"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
)

// MaxSlotBuffer specifies the max buffer given to slots from
// incoming objects. (24 mins with mainnet spec)
const MaxSlotBuffer = uint64(1 << 7)

// StartTime returns the start time in terms of its unix epoch
// value.
func StartTime(genesis uint64, slot types.Slot) time.Time {
	duration := time.Second * time.Duration(slot.Mul(params.BeaconConfig().SecondsPerSlot))
	startTime := time.Unix(int64(genesis), 0).Add(duration) // lint:ignore uintcast -- Genesis timestamp will not exceed int64 in your lifetime.
	return startTime
}

// SinceGenesis returns the number of slots since
// the provided genesis time.
func SinceGenesis(genesis time.Time) types.Slot {
	if genesis.After(prysmTime.Now()) { // Genesis has not occurred yet.
		return 0
	}
	return types.Slot(uint64(prysmTime.Since(genesis).Seconds()) / params.BeaconConfig().SecondsPerSlot)
}

// EpochsSinceGenesis returns the number of epochs since
// the provided genesis time.
func EpochsSinceGenesis(genesis time.Time) types.Epoch {
	return types.Epoch(SinceGenesis(genesis) / params.BeaconConfig().SlotsPerEpoch)
}

// DivideSlotBy divides the SECONDS_PER_SLOT configuration
// parameter by a specified number. It returns a value of time.Duration
// in milliseconds, useful for dividing values such as 1 second into
// millisecond-based durations.
func DivideSlotBy(timesPerSlot int64) time.Duration {
	return time.Duration(int64(params.BeaconConfig().SecondsPerSlot*1000)/timesPerSlot) * time.Millisecond
}

// MultiplySlotBy multiplies the SECONDS_PER_SLOT configuration
// parameter by a specified number. It returns a value of time.Duration
// in millisecond-based durations.
func MultiplySlotBy(times int64) time.Duration {
	return time.Duration(int64(params.BeaconConfig().SecondsPerSlot)*times) * time.Second
}

// AbsoluteValueSlotDifference between two slots.
func AbsoluteValueSlotDifference(x, y types.Slot) uint64 {
	if x > y {
		return uint64(x.SubSlot(y))
	}
	return uint64(y.SubSlot(x))
}

// ToEpoch returns the epoch number of the input slot.
//
// Spec pseudocode definition:
//  def compute_epoch_at_slot(slot: Slot) -> Epoch:
//    """
//    Return the epoch number at ``slot``.
//    """
//    return Epoch(slot // SLOTS_PER_EPOCH)
func ToEpoch(slot types.Slot) types.Epoch {
	return types.Epoch(slot.DivSlot(params.BeaconConfig().SlotsPerEpoch))
}

// EpochStart returns the first slot number of the
// current epoch.
//
// Spec pseudocode definition:
//  def compute_start_slot_at_epoch(epoch: Epoch) -> Slot:
//    """
//    Return the start slot of ``epoch``.
//    """
//    return Slot(epoch * SLOTS_PER_EPOCH)
func EpochStart(epoch types.Epoch) (types.Slot, error) {
	slot, err := params.BeaconConfig().SlotsPerEpoch.SafeMul(uint64(epoch))
	if err != nil {
		return slot, errors.Errorf("start slot calculation overflows: %v", err)
	}
	return slot, nil
}

// EpochEnd returns the last slot number of the
// current epoch.
func EpochEnd(epoch types.Epoch) (types.Slot, error) {
	if epoch == math.MaxUint64 {
		return 0, errors.New("start slot calculation overflows")
	}
	slot, err := EpochStart(epoch + 1)
	if err != nil {
		return 0, err
	}
	return slot - 1, nil
}

// IsEpochStart returns true if the given slot number is an epoch starting slot
// number.
func IsEpochStart(slot types.Slot) bool {
	return slot%params.BeaconConfig().SlotsPerEpoch == 0
}

// IsEpochEnd returns true if the given slot number is an epoch ending slot
// number.
func IsEpochEnd(slot types.Slot) bool {
	return IsEpochStart(slot + 1)
}

// SinceEpochStarts returns number of slots since the start of the epoch.
func SinceEpochStarts(slot types.Slot) types.Slot {
	return slot % params.BeaconConfig().SlotsPerEpoch
}

// VerifyTime validates the input slot is not from the future.
func VerifyTime(genesisTime uint64, slot types.Slot, timeTolerance time.Duration) error {
	slotTime, err := ToTime(genesisTime, slot)
	if err != nil {
		return err
	}

	// Defensive check to ensure unreasonable slots are rejected
	// straight away.
	if err := ValidateClock(slot, genesisTime); err != nil {
		return err
	}

	currentTime := prysmTime.Now()
	diff := slotTime.Sub(currentTime)

	if diff > timeTolerance {
		return fmt.Errorf("could not process slot from the future, slot time %s > current time %s", slotTime, currentTime)
	}
	return nil
}

// ToTime takes the given slot and genesis time to determine the start time of the slot.
func ToTime(genesisTimeSec uint64, slot types.Slot) (time.Time, error) {
	timeSinceGenesis, err := slot.SafeMul(params.BeaconConfig().SecondsPerSlot)
	if err != nil {
		return time.Unix(0, 0), fmt.Errorf("slot (%d) is in the far distant future: %w", slot, err)
	}
	sTime, err := timeSinceGenesis.SafeAdd(genesisTimeSec)
	if err != nil {
		return time.Unix(0, 0), fmt.Errorf("slot (%d) is in the far distant future: %w", slot, err)
	}
	return time.Unix(int64(sTime), 0), nil // lint:ignore uintcast -- A timestamp will not exceed int64 in your lifetime.
}

// Since computes the number of time slots that have occurred since the given timestamp.
func Since(time time.Time) types.Slot {
	return CurrentSlot(uint64(time.Unix()))
}

// CurrentSlot returns the current slot as determined by the local clock and
// provided genesis time.
func CurrentSlot(genesisTimeSec uint64) types.Slot {
	now := uint64(prysmTime.Now().Unix())
	if now < genesisTimeSec {
		return 0
	}
	return types.Slot((now - genesisTimeSec) / params.BeaconConfig().SecondsPerSlot)
}

// ValidateClock validates a provided slot against the local
// clock to ensure slots that are unreasonable are returned with
// an error.
func ValidateClock(slot types.Slot, genesisTimeSec uint64) error {
	maxPossibleSlot := CurrentSlot(genesisTimeSec).Add(MaxSlotBuffer)
	// Defensive check to ensure that we only process slots up to a hard limit
	// from our local clock.
	if slot > maxPossibleSlot {
		return fmt.Errorf("slot %d > %d which exceeds max allowed value relative to the local clock", slot, maxPossibleSlot)
	}
	return nil
}

// RoundUpToNearestEpoch rounds up the provided slot value to the nearest epoch.
func RoundUpToNearestEpoch(slot types.Slot) types.Slot {
	if slot%params.BeaconConfig().SlotsPerEpoch != 0 {
		slot -= slot % params.BeaconConfig().SlotsPerEpoch
		slot += params.BeaconConfig().SlotsPerEpoch
	}
	return slot
}

// VotingPeriodStartTime returns the current voting period's start time
// depending on the provided genesis and current slot.
func VotingPeriodStartTime(genesis uint64, slot types.Slot) uint64 {
	slots := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))
	startTime := uint64((slot - slot.ModSlot(slots)).Mul(params.BeaconConfig().SecondsPerSlot))
	return genesis + startTime
}

// PrevSlot returns previous slot, with an exception in slot 0 to prevent underflow.
func PrevSlot(slot types.Slot) types.Slot {
	if slot > 0 {
		return slot.Sub(1)
	}
	return 0
}

// SyncCommitteePeriod returns the sync committee period of input epoch `e`.
//
// Spec code:
// def compute_sync_committee_period(epoch: Epoch) -> uint64:
//    return epoch // EPOCHS_PER_SYNC_COMMITTEE_PERIOD
func SyncCommitteePeriod(e types.Epoch) uint64 {
	return uint64(e / params.BeaconConfig().EpochsPerSyncCommitteePeriod)
}

// SyncCommitteePeriodStartEpoch returns the start epoch of a sync committee period.
func SyncCommitteePeriodStartEpoch(e types.Epoch) (types.Epoch, error) {
	// Overflow is impossible here because of division of `EPOCHS_PER_SYNC_COMMITTEE_PERIOD`.
	startEpoch, err := mathutil.Mul64(SyncCommitteePeriod(e), uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod))
	if err != nil {
		return 0, err
	}
	return types.Epoch(startEpoch), nil
}
