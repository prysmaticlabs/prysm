package kv

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AttCaches defines the caches used to satisfy attestation pool interface.
// These caches are KV store for various attestations
// such are unaggregated, aggregated or attestations within a block.
type AttCaches struct {
	aggregatedAtt   *cache.Cache
	unAggregatedAtt *cache.Cache
	forkchoiceAtt   *cache.Cache
	blockAtt        *cache.Cache
}

// NewAttCaches initializes a new attestation pool consists of multiple KV store in cache for
// various kind of attestations.
func NewAttCaches() *AttCaches {
	secsInEpoch := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)

	// Create caches with default expiration time of one epoch and which
	// purges expired items every epoch.
	pool := &AttCaches{
		unAggregatedAtt: cache.New(secsInEpoch*time.Second, secsInEpoch*time.Second),
		aggregatedAtt:   cache.New(secsInEpoch*time.Second, secsInEpoch*time.Second),
		forkchoiceAtt:   cache.New(secsInEpoch*time.Second, secsInEpoch*time.Second),
		blockAtt:        cache.New(secsInEpoch*time.Second, secsInEpoch*time.Second),
	}

	return pool
}
