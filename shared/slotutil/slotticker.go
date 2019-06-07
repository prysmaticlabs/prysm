package slotutil

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// SlotTicker is a special ticker for the beacon chain block.
// The channel emits over the slot interval, and ensures that
// the ticks are in line with the genesis time. This means that
// the duration between the ticks and the genesis time are always a
// multiple of the slot duration.
// In addition, the channel returns the new slot number.
type SlotTicker struct {
	c    chan uint64
	done chan struct{}
}

// C returns the ticker channel. Call Cancel afterwards to ensure
// that the goroutine exits cleanly.
func (s *SlotTicker) C() <-chan uint64 {
	return s.c
}

// Done should be called to clean up the ticker.
func (s *SlotTicker) Done() {
	go func() {
		s.done <- struct{}{}
	}()
}

// GetSlotTicker is the constructor for SlotTicker.
func GetSlotTicker(genesisTime time.Time, secondsPerSlot uint64) *SlotTicker {
	ticker := &SlotTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime, secondsPerSlot, time.Since, time.Until, time.After)
	return ticker
}

func (s *SlotTicker) start(
	genesisTime time.Time,
	secondsPerSlot uint64,
	since func(time.Time) time.Duration,
	until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {

	d := time.Duration(secondsPerSlot) * time.Second

	go func() {
		sinceGenesis := since(genesisTime)

		var nextTickTime time.Time
		var slot uint64
		if sinceGenesis < 0 {
			// Handle when the current time is before the genesis time.
			nextTickTime = genesisTime
			slot = params.BeaconConfig().GenesisSlot
		} else {
			nextTick := sinceGenesis.Truncate(d) + d
			nextTickTime = genesisTime.Add(nextTick)
			slot = uint64(nextTick/d) + params.BeaconConfig().GenesisSlot
		}

		for {
			waitTime := until(nextTickTime)
			select {
			case <-after(waitTime):
				s.c <- slot
				slot++
				nextTickTime = nextTickTime.Add(d)
			case <-s.done:
				return
			}
		}
	}()
}
