// Package kv includes a key-value store implementation
// of an attestation cache used to satisfy important use-cases
// such as aggregation in a beacon node runtime.
package kv

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
)

// AttCaches defines the caches used to satisfy attestation pool interface.
// These caches are KV store for various attestations
// such are unaggregated, aggregated or attestations within a block.
type AttCaches struct {
	aggregatedAttLock  sync.RWMutex
	aggregatedAtt      map[attestation.Id][]ethpb.Att
	unAggregateAttLock sync.RWMutex
	unAggregatedAtt    map[attestation.Id]ethpb.Att
	forkchoiceAttLock  sync.RWMutex
	forkchoiceAtt      map[attestation.Id]ethpb.Att
	blockAttLock       sync.RWMutex
	blockAtt           map[attestation.Id][]ethpb.Att
	seenAtt            *cache.Cache
}

// NewAttCaches initializes a new attestation pool consists of multiple KV store in cache for
// various kind of attestations.
func NewAttCaches() *AttCaches {
	secsInEpoch := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	c := cache.New(2*secsInEpoch*time.Second, 2*secsInEpoch*time.Second)
	pool := &AttCaches{
		unAggregatedAtt: make(map[attestation.Id]ethpb.Att),
		aggregatedAtt:   make(map[attestation.Id][]ethpb.Att),
		forkchoiceAtt:   make(map[attestation.Id]ethpb.Att),
		blockAtt:        make(map[attestation.Id][]ethpb.Att),
		seenAtt:         c,
	}

	return pool
}
