package attestation

import (
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// The max amount of unaggregated attestation a node can receive in one epoch.
// Bounded by the validators can participate in eth2.
var unaggregatedCacheSize = int64(params.BeaconConfig().ValidatorRegistryLimit)
// The max amount of aggregated attestation a node can receive in one epoch.
// Bounded by the max committee count in one epoch.
var aggregatedCacheSize = int64(params.BeaconConfig().MaxCommitteesPerSlot * params.BeaconConfig().SlotsPerEpoch)

// Pool defines an implementation of the attestation pool interface
// using cache as underlying kv store for various incoming attestations
// such are unaggregated, aggregated or within a block.
type Pool struct {
	aggregatedAtt   *ccache.Cache
	unAggregatedAtt *ccache.Cache
	attInBlock      *ccache.Cache
}

// NewPool initializes a new attestation pool consists of multiple KV store in cache for
// various kind of aggregations.
func NewPool() *Pool {

	pool := &Pool{
		unAggregatedAtt:          ccache.New(ccache.Configure().MaxSize(unaggregatedCacheSize)),
		aggregatedAtt:          ccache.New(ccache.Configure().MaxSize(aggregatedCacheSize)),
		attInBlock: ccache.New(ccache.Configure().MaxSize(aggregatedCacheSize)),
	}

	return pool
}
