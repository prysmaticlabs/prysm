package params

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/pkg/errors"
)

// SlotScheduleEntry is one slot duration era, starting at the given epoch.
type SlotScheduleEntry struct {
	Epoch              primitives.Epoch
	SlotDurationMillis uint64
}

// SlotSchedule lists slot duration eras in increasing epoch order, starting at the genesis epoch.
type SlotSchedule []SlotScheduleEntry

const (
	// maxMs bounds millisecond totals so their time.Duration conversion cannot overflow int64.
	maxMs = uint64(math.MaxInt64 / int64(time.Millisecond))

	// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R120
	slotDurationGranularityMillis = 1000
)

// SlotComponent names an intra-slot deadline.
type SlotComponent uint8

const (
	ProposerReorgCutoff SlotComponent = iota
	AttestationDue
	AggregateDue
	SyncMessageDue
	ContributionDue
	PayloadDue
	PayloadAttestationDue
)

// SinceGenesis returns the duration from genesis to the start of the given slot.
func (s SlotSchedule) SinceGenesis(slot primitives.Slot, slotsPerEpoch primitives.Slot) (time.Duration, error) {
	timeMs, err := s.timeAtSlotMs(0, slot, slotsPerEpoch)
	if err != nil || timeMs > maxMs {
		return 0, errors.Errorf("slot %d is in the far distant future", slot)
	}

	return time.Duration(timeMs) * time.Millisecond, nil
}

// SlotAt returns the slot at the given time.
func (s SlotSchedule) SlotAt(genesis, tm time.Time, slotsPerEpoch primitives.Slot) primitives.Slot {
	if tm.Before(genesis) {
		return 0
	}

	// Slot durations are whole milliseconds, so flooring the elapsed time to milliseconds does not change the slot.
	return s.slotAtTimeMs(0, uint64(tm.Sub(genesis).Milliseconds()), slotsPerEpoch)
}

// durationAt returns the duration of the given slot.
func (s SlotSchedule) durationAt(slot primitives.Slot, slotsPerEpoch primitives.Slot) time.Duration {
	return time.Duration(s.slotDurationMs(computeEpochAtSlot(slot, slotsPerEpoch))) * time.Millisecond
}

// slotDurationMs returns the slot duration in effect at the given epoch, in milliseconds.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R319
func (s SlotSchedule) slotDurationMs(epoch primitives.Epoch) uint64 {
	slotDurationMs := uint64(0)
	for _, entry := range s {
		if epoch >= entry.Epoch {
			slotDurationMs = entry.SlotDurationMillis
		}
	}

	return slotDurationMs
}

// timeAtSlotMs returns the Unix time in milliseconds at the start of the given slot.
// It returns an error if the time does not fit in an uint64.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R332
func (s SlotSchedule) timeAtSlotMs(genesisTimeMs uint64, slot, slotsPerEpoch primitives.Slot) (uint64, error) {
	endSlot := slot
	timeMs := genesisTimeMs
	for _, entry := range slices.Backward(s) {
		forkSlot := computeStartSlotAtEpoch(entry.Epoch, slotsPerEpoch)
		if forkSlot < endSlot {
			slots := endSlot - forkSlot
			ms, err := slots.SafeMul(entry.SlotDurationMillis)
			if err != nil || uint64(ms) > math.MaxUint64-timeMs {
				return 0, errors.Errorf("time at slot %d overflows", slot)
			}

			timeMs += uint64(ms)
			endSlot = forkSlot
		}
	}

	return timeMs, nil
}

// slotAtTimeMs returns the slot at the given Unix time in milliseconds.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R351
func (s SlotSchedule) slotAtTimeMs(genesisTimeMs, timeMs uint64, slotsPerEpoch primitives.Slot) primitives.Slot {
	if timeMs < genesisTimeMs {
		return 0
	}

	startSlot, startTimeMs, slotDurationMs := primitives.Slot(0), genesisTimeMs, uint64(0)
	for _, entry := range s {
		forkSlot := computeStartSlotAtEpoch(entry.Epoch, slotsPerEpoch)
		forkTimeMs, err := s.timeAtSlotMs(genesisTimeMs, forkSlot, slotsPerEpoch)
		if err != nil {
			// This fork and the next ones start too late to be represented, so after the given time.
			break
		}

		if timeMs >= forkTimeMs {
			startSlot = forkSlot
			startTimeMs = forkTimeMs
			slotDurationMs = entry.SlotDurationMillis
		}
	}

	timeDiffMs := timeMs - startTimeMs
	slots := primitives.Slot(timeDiffMs / slotDurationMs)
	return startSlot + slots
}

// computeStartSlotAtEpoch returns the start slot of the given epoch.
// It assumes the product does not overflow, which validateSlotDurations ensures for slot duration eras.
func computeStartSlotAtEpoch(epoch primitives.Epoch, slotsPerEpoch primitives.Slot) primitives.Slot {
	return primitives.Slot(uint64(epoch) * uint64(slotsPerEpoch))
}

// computeEpochAtSlot returns the epoch of the given slot.
func computeEpochAtSlot(slot, slotsPerEpoch primitives.Slot) primitives.Epoch {
	return primitives.Epoch(slot / slotsPerEpoch)
}

// SlotDurations returns the slot duration eras derived from the fork configuration, skipping unscheduled forks.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R298
func (b *BeaconChainConfig) SlotDurations() SlotSchedule {
	schedule := SlotSchedule{{Epoch: b.GenesisEpoch, SlotDurationMillis: b.SlotDurationMillis()}}
	if b.HezeForkEpoch != b.FarFutureEpoch {
		schedule = append(schedule, SlotScheduleEntry{Epoch: b.HezeForkEpoch, SlotDurationMillis: b.SlotDurationMillisHeze})
	}

	return schedule
}

// HasSlotDurationChange reports whether a fork changing the slot duration is scheduled.
func (b *BeaconChainConfig) HasSlotDurationChange() bool {
	return len(b.SlotDurations()) > 1
}

// ShortestSlotDuration returns the shortest scheduled slot duration.
func (b *BeaconChainConfig) ShortestSlotDuration() time.Duration {
	shortest := uint64(math.MaxUint64)
	for _, e := range b.SlotDurations() {
		shortest = min(shortest, e.SlotDurationMillis)
	}

	return time.Duration(shortest) * time.Millisecond
}

// LongestSlotDuration returns the longest scheduled slot duration.
func (b *BeaconChainConfig) LongestSlotDuration() time.Duration {
	longest := uint64(0)
	for _, e := range b.SlotDurations() {
		longest = max(longest, e.SlotDurationMillis)
	}

	return time.Duration(longest) * time.Millisecond
}

// ShortestEpochDuration returns the duration of one epoch at the shortest scheduled slot duration.
func (b *BeaconChainConfig) ShortestEpochDuration() time.Duration {
	return time.Duration(b.SlotsPerEpoch) * b.ShortestSlotDuration()
}

// LongestEpochDuration returns the duration of one epoch at the longest scheduled slot duration.
func (b *BeaconChainConfig) LongestEpochDuration() time.Duration {
	return time.Duration(b.SlotsPerEpoch) * b.LongestSlotDuration()
}

// SinceGenesis returns the duration from genesis to the start of the given slot.
func (b *BeaconChainConfig) SinceGenesis(slot primitives.Slot) (time.Duration, error) {
	return b.SlotDurations().SinceGenesis(slot, b.SlotsPerEpoch)
}

// SlotDurationAt returns the duration of the given slot.
func (b *BeaconChainConfig) SlotDurationAt(slot primitives.Slot) time.Duration {
	return b.SlotDurations().durationAt(slot, b.SlotsPerEpoch)
}

// SlotDurationMillisAtEpoch returns the slot duration in effect at the given epoch, in milliseconds.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-d86ff035f381a7179bbf845cda1b9d9ced9e070ee8cd5501dab6fc79ba233051R319
func (b *BeaconChainConfig) SlotDurationMillisAtEpoch(epoch primitives.Epoch) uint64 {
	return b.SlotDurations().slotDurationMs(epoch)
}

// SlotFractionAt returns the given basis-point fraction of the slot's duration.
func (b *BeaconChainConfig) SlotFractionAt(bp primitives.BP, slot primitives.Slot) time.Duration {
	ms := uint64(bp) * uint64(b.SlotDurationAt(slot)/time.Millisecond) / uint64(BasisPoints)
	return time.Duration(ms) * time.Millisecond
}

// SlotComponentDurationAt returns the offset of the given deadline into the given slot.
// From Heze, the basis points apply to SLOT_DURATION_MS_HEZE, which is the duration of every Heze slot.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-2bd43b6a1876bf339bd0464c2ee6557f41c72eca7e57b8383aeeab962cd8b9ecR90
func (b *BeaconChainConfig) SlotComponentDurationAt(c SlotComponent, slot primitives.Slot) time.Duration {
	return b.SlotFractionAt(b.slotComponentBPS(c, slot), slot)
}

// BlobDataRetentionStartEpoch returns the earliest epoch of the data column serve range at the given epoch.
// Before Heze, the window is MIN_EPOCHS_FOR_DATA_COLUMN_SIDECARS_REQUESTS epochs. From Heze, that config value is
// deprecated and the window is MIN_BLOB_DATA_RETENTION_MS of wall-clock time across slot duration changes.
func (b *BeaconChainConfig) BlobDataRetentionStartEpoch(current primitives.Epoch) (primitives.Epoch, error) {
	if current < b.HezeForkEpoch {
		return epochsBefore(current, b.MinEpochsForDataColumnSidecarsRequest), nil
	}

	start, err := b.blobDataRetentionStartEpoch(current, b.MinBlobDataRetentionMillis)
	if err != nil {
		return 0, fmt.Errorf("blob data retention start epoch at epoch %d: %w", current, err)
	}

	return start, nil
}

// DataColumnRetentionStartEpoch returns the earliest epoch of the data column storage window at the given epoch.
// Before Heze, the window is the given number of epochs. From Heze, it is the longest of the serve range and the given
// number of epochs counted at SLOT_DURATION_MS, so that a retention configured in epochs keeps its wall-clock length
// across slot duration changes.
func (b *BeaconChainConfig) DataColumnRetentionStartEpoch(current, epochs primitives.Epoch) (primitives.Epoch, error) {
	if current < b.HezeForkEpoch {
		return epochsBefore(current, epochs), nil
	}

	start, err := b.BlobDataRetentionStartEpoch(current)
	if err != nil {
		return 0, fmt.Errorf("blob data retention start epoch: %w", err)
	}

	// A window of epochs too long to be expressed in milliseconds reaches back past genesis.
	baseMs := b.SlotDurationMillis()
	if uint64(epochs) > math.MaxUint64/uint64(b.SlotsPerEpoch)/baseMs {
		return b.GenesisEpoch, nil
	}

	epochsStart, err := b.blobDataRetentionStartEpoch(current, uint64(epochs)*uint64(b.SlotsPerEpoch)*baseMs)
	if err != nil {
		return 0, fmt.Errorf("retention start epoch of %d epochs at epoch %d: %w", epochs, current, err)
	}

	return min(start, epochsStart), nil
}

// blobDataRetentionStartEpoch returns the start epoch of the blob data retention window of the given length,
// preserving its wall-clock length across slot duration changes.
// It returns an error if the start of the given epoch does not fit in uint64 milliseconds.
// https://github.com/ethereum/consensus-specs/pull/5718/files#diff-021eb38195386e46ad74febe9ccad8bb62eed4248f49ff998be1cb73d69a5955R167
func (b *BeaconChainConfig) blobDataRetentionStartEpoch(epoch primitives.Epoch, windowMs uint64) (primitives.Epoch, error) {
	s, slotsPerEpoch := b.SlotDurations(), b.SlotsPerEpoch
	if uint64(epoch) > math.MaxUint64/uint64(slotsPerEpoch) {
		return 0, errors.Errorf("start slot of epoch %d overflows", epoch)
	}

	currentStartSlot := computeStartSlotAtEpoch(epoch, slotsPerEpoch)
	currentStartMs, err := s.timeAtSlotMs(0, currentStartSlot, slotsPerEpoch)
	if err != nil {
		return 0, fmt.Errorf("time at slot %d: %w", currentStartSlot, err)
	}

	if currentStartMs < windowMs {
		return b.GenesisEpoch, nil
	}

	windowStartMs := currentStartMs - windowMs
	return computeEpochAtSlot(s.slotAtTimeMs(0, windowStartMs, slotsPerEpoch), slotsPerEpoch), nil
}

// epochsBefore returns the given number of epochs before current, floored at the genesis epoch.
func epochsBefore(current, epochs primitives.Epoch) primitives.Epoch {
	if current > epochs {
		return current - epochs
	}

	return 0
}

// validateSlotDurations checks the slot durations derived from the fork configuration.
func (b *BeaconChainConfig) validateSlotDurations() error {
	if b.SlotsPerEpoch == 0 {
		return errors.New("slots per epoch must be positive")
	}

	for _, e := range b.SlotDurations()[1:] {
		if e.SlotDurationMillis == 0 || e.SlotDurationMillis%slotDurationGranularityMillis != 0 {
			return errors.Errorf("slot duration %dms at epoch %d must be a positive multiple of %d", e.SlotDurationMillis, e.Epoch, slotDurationGranularityMillis)
		}

		if uint64(e.Epoch) > math.MaxUint64/uint64(b.SlotsPerEpoch) {
			return errors.Errorf("slot duration change at epoch %d overflows slot arithmetic", e.Epoch)
		}
	}

	return nil
}

func (b *BeaconChainConfig) slotComponentBPS(c SlotComponent, slot primitives.Slot) primitives.BP {
	postGloas := primitives.Epoch(slot.DivSlot(b.SlotsPerEpoch)) >= b.GloasForkEpoch
	switch c {
	case ProposerReorgCutoff:
		return b.ProposerReorgCutoffBPS
	case AttestationDue:
		if postGloas {
			return b.AttestationDueBPSGloas
		}
		return b.AttestationDueBPS
	case AggregateDue:
		if postGloas {
			return b.AggregateDueBPSGloas
		}
		return b.AggregateDueBPS
	case SyncMessageDue:
		if postGloas {
			return b.SyncMessageDueBPSGloas
		}
		return b.SyncMessageDueBPS
	case ContributionDue:
		if postGloas {
			return b.ContributionDueBPSGloas
		}
		return b.ContributionDueBPS
	case PayloadDue:
		return b.PayloadDueBPS
	case PayloadAttestationDue:
		return b.PayloadAttestationDueBPS
	default:
		return 0
	}
}
