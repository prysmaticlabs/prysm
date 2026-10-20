package helpers

import (
	"time"

	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	prysmTime "github.com/OffchainLabs/prysm/v7/time"
	"github.com/OffchainLabs/prysm/v7/time/slots"
)

// EpochTicker is a special ticker for timing epoch changes.
// The channel emits once per epoch, half a slot past the middle of the epoch, in order to keep results consistent.
// The tick times follow the slot durations derived from the fork configuration.
// In addition, the channel returns the epoch number.
type EpochTicker struct {
	c    chan uint64
	done chan struct{}
}

// C returns the ticker channel. Call Cancel afterwards to ensure
// that the goroutine exits cleanly.
func (s *EpochTicker) C() <-chan uint64 {
	return s.c
}

// Done should be called to clean up the ticker.
func (s *EpochTicker) Done() {
	go func() {
		s.done <- struct{}{}
	}()
}

// NewEpochTicker starts the EpochTicker.
func NewEpochTicker(genesisTime time.Time) *EpochTicker {
	ticker := &EpochTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime, prysmTime.Now, prysmTime.Until, time.After)
	return ticker
}

func (s *EpochTicker) start(
	genesisTime time.Time,
	now func() time.Time,
	until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {

	go func() {
		epoch := slots.ToEpoch(slots.At(genesisTime, now()))
		if epochTickTime(genesisTime, epoch).Before(now()) {
			epoch++
		}

		for {
			select {
			case <-after(until(epochTickTime(genesisTime, epoch))):
				s.c <- uint64(epoch)
				epoch++
			case <-s.done:
				return
			}
		}
	}()
}

// epochTickTime returns the time half a slot past the middle of the given epoch.
func epochTickTime(genesisTime time.Time, epoch primitives.Epoch) time.Time {
	middleSlot := slots.UnsafeEpochStart(epoch) + params.BeaconConfig().SlotsPerEpoch/2
	return slots.UnsafeStartTime(genesisTime, middleSlot).Add(params.BeaconConfig().SlotDurationAt(middleSlot) / 2)
}
