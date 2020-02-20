package state

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
)

// skipSlotCache exists for the unlikely scenario that is a large gap between the head state and
// the current slot. If the beacon chain were ever to be stalled for several epochs, it may be
// difficult or impossible to compute the appropriate beacon state for assignments within a
// reasonable amount of time.
var skipSlotCache = cache.NewSkipSlotCache()
