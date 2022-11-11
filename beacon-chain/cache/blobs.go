package cache

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
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
	cache map[types.Slot]*eth.BlobsSidecar
	sync.Mutex
}

func NewBlobsCache() *BlobsCache {
	return &BlobsCache{
		cache: map[types.Slot]*eth.BlobsSidecar{},
	}
}

func (b *BlobsCache) Get(slot types.Slot) (*eth.BlobsSidecar, error) {
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

func (b *BlobsCache) Put(sidecar *eth.BlobsSidecar) {
	b.Lock()
	defer b.Unlock()
	b.cache[sidecar.BeaconBlockSlot] = sidecar
}
