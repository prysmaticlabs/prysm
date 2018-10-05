package utils

import (
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/params"
)

// CurrentSlot returns slot number based on the genesis timestamp.
func CurrentSlot(genesisTime time.Time) uint64 {
	secondsSinceGenesis := uint64(time.Since(genesisTime).Seconds())
	currentSlot := secondsSinceGenesis / params.GetConfig().SlotDuration
	if currentSlot < 1 {
		return 0
	}
	return currentSlot - 1
}
