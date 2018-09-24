package utils

import (
	"math"
	"time"
)

// GenesisTime used by the protocol.
var GenesisTime = time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC) // September 2018

// Clock represents a time providing interface that can be mocked for testing.
type Clock interface {
	Now() time.Time
}

// RealClock represents an unmodified clock.
type RealClock struct{}

// Now represents the standard functionality of time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// CurrentBeaconSlot based on the seconds since genesis.
func CurrentBeaconSlot() uint64 {
	secondsSinceGenesis := time.Since(GenesisTime).Seconds()
	return uint64(math.Floor(secondsSinceGenesis / 8.0))
}

// WaitUntilTimestamp sleeps until a specific time is reached after
// a certain duratio. For example, if the genesis block
// was at 12:00:00PM and the current time is 12:00:03PM,
// we want the next slot to tick at 12:00:08PM so we can use
// this helper method to achieve that purpose.
func WaitUntilTimestamp(duration time.Duration) {
	d := time.Until(time.Now().Add(duration).Truncate(duration))
	time.Sleep(d)
}
