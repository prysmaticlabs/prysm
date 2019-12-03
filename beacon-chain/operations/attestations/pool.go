package attestations

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Pool defines an implementation of the attestation pool interface
// using cache as underlying kv store for various incoming attestations
// such are unaggregated, aggregated or within a block.
type Pool struct {
	aggregatedAtt   *cache.Cache
	unAggregatedAtt *cache.Cache
	attInBlock      *cache.Cache
}

// NewPool initializes a new attestation pool consists of multiple KV store in cache for
// various kind of aggregations.
func NewPool() *Pool {

	secsInEpoch := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)

	// Create caches with default expiration time of one epoch and which
	// purges expired items every other epoch.
	pool := &Pool{
		unAggregatedAtt: cache.New(secsInEpoch/time.Minute, 2*secsInEpoch/time.Minute),
		aggregatedAtt:   cache.New(secsInEpoch/time.Minute, 2*secsInEpoch/time.Minute),
		attInBlock:      cache.New(secsInEpoch/time.Minute, 2*secsInEpoch/time.Minute),
	}

	return pool
}
