package slotutil

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// SlotStartTime returns the start time in terms of its unix epoch
// value.
func SlotStartTime(genesis uint64, slot uint64) time.Time {
	duration := time.Second * time.Duration(slot*params.BeaconConfig().SecondsPerSlot)
	startTime := time.Unix(int64(genesis), 0).Add(duration)
	return startTime
}

// SlotsSinceGenesis returns the number of slots since
// the provided genesis time.
func SlotsSinceGenesis(genesis time.Time) uint64 {
	return uint64(roughtime.Since(genesis).Seconds()) / params.BeaconConfig().SecondsPerSlot
}

// EpochsSinceGenesis returns the number of slots since
// the provided genesis time.
func EpochsSinceGenesis(genesis time.Time) uint64 {
	return SlotsSinceGenesis(genesis) / params.BeaconConfig().SlotsPerEpoch
}
