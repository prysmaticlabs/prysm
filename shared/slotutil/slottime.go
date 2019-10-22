package slotutil

import (
	"time"

	"github.com/prysmaticlabs/prysm/shared/params"
)

// SlotStartTime returns the start time in terms of its unix epoch
// value.
func SlotStartTime(genesis uint64, slot uint64) time.Time {
	duration := time.Second * time.Duration(slot*params.BeaconConfig().SecondsPerSlot)
	startTime := time.Unix(int64(genesis), 0).Add(duration)
	return startTime
}
