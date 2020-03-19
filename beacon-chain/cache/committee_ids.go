package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type committeeIDs struct {
	cache *lru.Cache
	lock  sync.RWMutex
}

// CommitteeIDs for attestations.
var CommitteeIDs = newCommitteeIDs()

func newCommitteeIDs() *committeeIDs {
	cache, err := lru.New(8)
	if err != nil {
		panic(err)
	}
	return &committeeIDs{cache: cache}
}

// AddIDs to the cache for attestation committees by epoch.
func (t *committeeIDs) AddIDs(indices []uint64, epoch uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	val, exists := t.cache.Get(epoch)
	if exists {
		indices = sliceutil.UnionUint64(append(indices, val.([]uint64)...))
	}
	t.cache.Add(epoch, indices)
}

// GetIDs from the cache for attestation committees by epoch.
func (t *committeeIDs) GetIDs(epoch uint64) []uint64 {
	val, exists := t.cache.Get(epoch)
	if !exists {
		return []uint64{}
	}
	return val.([]uint64)
}
