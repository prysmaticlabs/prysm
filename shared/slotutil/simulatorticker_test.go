package slotutil

import (
	"testing"
	"time"
)

func TestSimulatorTicker(t *testing.T) {
	ticker := SimulatorTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	defer ticker.Done()

	var sinceDuration time.Duration
	since := func(time.Time) time.Duration {
		return sinceDuration
	}

	var untilDuration time.Duration
	until := func(time.Time) time.Duration {
		return untilDuration
	}

	var tick chan time.Time
	after := func(time.Duration) <-chan time.Time {
		return tick
	}

	genesisTime := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	slotDuration := uint64(8)

	// Test when the ticker starts immediately after genesis time.
	sinceDuration = 1 * time.Second
	untilDuration = 7 * time.Second
	// Send in a non-zero slot
	currentslot := uint64(8)
	// Make this a buffered channel to prevent a deadlock since
	// the other goroutine calls a function in this goroutine.
	tick = make(chan time.Time, 2)
	ticker.start(genesisTime, slotDuration, currentslot, since, until, after)

	// Tick once
	tick <- time.Now()
	slot := <-ticker.C()
	if slot != 9 {
		t.Fatalf("Expected 9, got %d", slot)
	}

	// Tick twice
	tick <- time.Now()
	slot = <-ticker.C()
	if slot != 10 {
		t.Fatalf("Expected 10, got %d", slot)
	}
}
