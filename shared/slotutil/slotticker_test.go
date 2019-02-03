package slotutil

import (
	"testing"
	"time"
)

func TestSlotTicker(t *testing.T) {
	ticker := &SlotTicker{
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
	// Make this a buffered channel to prevent a deadlock since
	// the other goroutine calls a function in this goroutine.
	tick = make(chan time.Time, 2)
	ticker.start(genesisTime, slotDuration, since, until, after)

	// Tick once
	tick <- time.Now()
	slot := <-ticker.C()
	if slot != 1 {
		t.Fatalf("Expected 1, got %d", slot)
	}

	// Tick twice
	tick <- time.Now()
	slot = <-ticker.C()
	if slot != 2 {
		t.Fatalf("Expected 2, got %d", slot)
	}
}

func TestSlotTickerGenesis(t *testing.T) {
	ticker := &SlotTicker{
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

	// Test when the ticker starts before genesis time.
	sinceDuration = -1 * time.Second
	untilDuration = 1 * time.Second
	// Make this a buffered channel to prevent a deadlock since
	// the other goroutine calls a function in this goroutine.
	tick = make(chan time.Time, 2)
	ticker.start(genesisTime, slotDuration, since, until, after)

	// Tick once
	tick <- time.Now()
	slot := <-ticker.C()
	if slot != 0 {
		t.Fatalf("Expected 0, got %d", slot)
	}

	// Tick twice
	tick <- time.Now()
	slot = <-ticker.C()
	if slot != 1 {
		t.Fatalf("Expected 1, got %d", slot)
	}
}

func TestCurrentSlot(t *testing.T) {
	// Test slot 0
	genesisTime := time.Now()
	slot := CurrentSlot(genesisTime, 5, time.Since)
	if slot != 0 {
		t.Errorf("Expected 0, got: %d", slot)
	}

	// Test a future genesis time
	genesisTime = genesisTime.Add(3 * time.Second)
	slot = CurrentSlot(genesisTime, 5, time.Since)
	if slot != 0 {
		t.Errorf("Expected 0, got: %d", slot)
	}

	// Test slot 3
	genesisTime = genesisTime.Add(-18 * time.Second)
	slot = CurrentSlot(genesisTime, 5, time.Since)
	if slot != 3 {
		t.Errorf("Expected 3, got: %d", slot)
	}
}
