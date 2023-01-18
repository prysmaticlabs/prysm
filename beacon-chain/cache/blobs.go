package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	v1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
)

var (
	// blobsCacheMiss tracks the number of blobs requests that aren't present in the cache.
	blobsCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_blobs_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
	errNoSidecar = errors.New("no sidecar")
)

// BlobsCache caches the blobs for the proposer through life cycle.
type BlobsCache struct {
	cache map[types.Slot][]*v1.Blob
	sync.Mutex
}

func NewBlobsCache() *BlobsCache {
	return &BlobsCache{
		cache: map[types.Slot][]*v1.Blob{},
	}
}

func (b *BlobsCache) Get(slot types.Slot) ([]*v1.Blob, error) {
	b.Lock()
	defer b.Unlock()

	sc, ok := b.cache[slot]
	if !ok {
		blobsCacheMiss.Inc()
		return nil, errNoSidecar
	}
	delete(b.cache, slot)
	return sc, nil
}

func (b *BlobsCache) Put(slot types.Slot, blobs []*v1.Blob) {
	b.Lock()
	defer b.Unlock()
	b.cache[slot] = blobs
}
