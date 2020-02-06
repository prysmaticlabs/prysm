package cache

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type trackedCommitteeIndices struct {
	cache *lru.Cache
	lock sync.RWMutex
}

var TrackedCommitteeIndices = newTrackedCommitteeIndicies()

func newTrackedCommitteeIndicies() *trackedCommitteeIndices {
	cache, err := lru.New(8)
	if err != nil {
		panic(err)
	}
	return &trackedCommitteeIndices{cache:cache}
}

func (t *trackedCommitteeIndices) AddIndices(indices []uint64, epoch uint64) {
	t.lock.Lock()
	defer t.lock.Unlock()
	val, exists := t.cache.Get(epoch)
	if exists {
		indices = sliceutil.UnionUint64(append(indices, val.([]uint64)...))
	}
	t.cache.Add(epoch, indices)
}

func (t *trackedCommitteeIndices) GetIndices(epoch uint64) []uint64 {
	val, exists := t.cache.Get(epoch)
	if !exists {
		return []uint64{}
	}
	return val.([]uint64)
}
