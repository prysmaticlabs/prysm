package slotticker

import (
	"time"
)

// SimulatorTicker is modified version of the SlotTicker so that the
// simulator is able to generate blocks correctly even if a node is
// shutdown and started up again.
type SimulatorTicker struct {
	c    chan uint64
	done chan struct{}
}

// C returns the ticker channel. Call Cancel afterwards to ensure
// that the goroutine exits cleanly.
func (s *SimulatorTicker) C() <-chan uint64 {
	return s.c
}

// Done should be called to clean up the ticker.
func (s *SimulatorTicker) Done() {
	go func() {
		s.done <- struct{}{}
	}()
}

// GetSimulatorTicker is the constructor for SimulatorTicker.
func GetSimulatorTicker(genesisTime time.Time, slotDuration uint64) SimulatorTicker {
	ticker := SimulatorTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime, slotDuration, time.Since, time.Until, time.After)

	return ticker
}

// CurrentSimulatorSlot accepts the genesis time and returns the current time's slot.
func CurrentSimulatorSlot(
	genesisTime time.Time,
	slotDuration uint64,
	since func(time.Time) time.Duration) uint64 {

	sinceGenesis := since(genesisTime)
	if sinceGenesis < 0 {
		return 0
	}

	durationInSeconds := time.Duration(slotDuration) * time.Second
	return uint64(sinceGenesis / durationInSeconds)
}

func (s *SimulatorTicker) start(
	genesisTime time.Time,
	slotDuration uint64,
	since func(time.Time) time.Duration,
	until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {

	d := time.Duration(slotDuration) * time.Second

	go func() {
		sinceGenesis := since(genesisTime)

		var nextTickTime time.Time
		var slot uint64
		if sinceGenesis < 0 {
			// Handle when the current time is before the genesis time
			nextTickTime = genesisTime
			slot = 0
		} else {
			nextTick := sinceGenesis.Truncate(d) + d
			nextTickTime = genesisTime.Add(nextTick)
			slot = uint64(nextTick / d)
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
