package slotutil

import (
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
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

	// Tick once.
	tick <- time.Now()
	slot := <-ticker.C()
	if slot != 1+params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected %d, got %d", params.BeaconConfig().GenesisSlot+1, slot)
	}

	// Tick twice.
	tick <- time.Now()
	slot = <-ticker.C()
	if slot != 2+params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected %d, got %d", params.BeaconConfig().GenesisSlot+2, slot)
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

	// Tick once.
	tick <- time.Now()
	slot := <-ticker.C()
	if slot != params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected %d, got %d", params.BeaconConfig().GenesisSlot, slot)
	}

	// Tick twice.
	tick <- time.Now()
	slot = <-ticker.C()
	if slot != 1+params.BeaconConfig().GenesisSlot {
		t.Fatalf("Expected %d, got %d", params.BeaconConfig().GenesisSlot+1, slot)
	}
}

func TestCurrentSlot(t *testing.T) {
	// Test genesis slot
	genesisTime := time.Now()
	slotDurationSeconds := time.Second * time.Duration(params.BeaconConfig().SlotDuration)
	slot := CurrentSlot(genesisTime, params.BeaconConfig().SlotDuration, time.Since)
	if slot != params.BeaconConfig().GenesisSlot {
		t.Errorf("Expected %d, got: %d", params.BeaconConfig().GenesisSlot, slot)
	}

	// Test slot 3 after genesis.
	genesisTime = genesisTime.Add(slotDurationSeconds * 3)
	slot = CurrentSlot(genesisTime, params.BeaconConfig().SlotDuration, time.Since)
	if slot != 3*params.BeaconConfig().GenesisSlot {
		t.Errorf("Expected %d, got: %d", params.BeaconConfig().GenesisSlot*3, slot)
	}
}
