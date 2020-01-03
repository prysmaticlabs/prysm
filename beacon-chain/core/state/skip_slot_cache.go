package state

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// skipSlotCache exists for the unlikely scenario that is a large gap between the head state and
// the current slot. If the beacon chain were ever to be stalled for several epochs, it may be
// difficult or impossible to compute the appropriate beacon state for assignments within a
// reasonable amount of time.
var skipSlotCache, _ = lru.New(8)

var (
	skipSlotCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_hit",
		Help: "The total number of cache hits on the skip slot cache.",
	})
	skipSlotCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "skip_slot_cache_miss",
		Help: "The total number of cache misses on the skip slot cache.",
	})
)

func cacheKey(bState *pb.BeaconState) ([32]byte, error) {
	// the latest header has a zeroed 32 byte hash as the state root,
	// which isnt necessary for the purposes of making the cache key. As
	// the parent root and body root are sufficient to prevent any collisions.
	blockRoot, err := ssz.HashTreeRoot(bState.LatestBlockHeader)
	if err != nil {
		return [32]byte{}, err
	}
	return hashutil.FastSum256(append(bytesutil.Bytes8(bState.Slot), blockRoot[:]...)), nil
}
