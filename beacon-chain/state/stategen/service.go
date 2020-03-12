package stategen

import (
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// State represents a management object that handles the internal
// logic of maintaining both hot and cold states in DB.
type State struct {
	beaconDB                db.NoHeadAccessDatabase
	lastArchivedSlot        uint64
	slotsPerArchivedPoint   uint64
	epochBoundarySlotToRoot map[uint64][32]byte
	epochBoundaryLock       sync.RWMutex
	hotStateCache           *cache.HotStateCache
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB:                db,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
		hotStateCache:           cache.NewHotStateCache(),
	}
}

// This verifies the archive point frequency is valid. It checks the interval
// is a divisor of the number of slots per epoch. This ensures we have at least one
// archive point within range of our state root history when iterating
// backwards. It also ensures the archive points align with hot state summaries
// which makes it quicker to migrate hot to cold.
func verifySlotsPerArchivePoint(slotsPerArchivePoint uint64) bool {
	return slotsPerArchivePoint > 0 &&
		slotsPerArchivePoint%params.BeaconConfig().SlotsPerEpoch == 0
}
