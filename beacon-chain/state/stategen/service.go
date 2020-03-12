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
	splitInfo               *splitSlotAndRoot
}

// This tracks the split point. The point where slot and the block root of
// cold and hot sections of the DB splits.
type splitSlotAndRoot struct {
	slot uint64
	root [32]byte
}

// New returns a new state management object.
func New(db db.NoHeadAccessDatabase) *State {
	return &State{
		beaconDB:                db,
		epochBoundarySlotToRoot: make(map[uint64][32]byte),
		hotStateCache:           cache.NewHotStateCache(),
		splitInfo:               &splitSlotAndRoot{slot: 0, root: params.BeaconConfig().ZeroHash},
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
