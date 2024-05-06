// Package kv includes a key-value store implementation
// of an attestation cache used to satisfy important use-cases
// such as aggregation in a beacon node runtime.
package kv

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

var hashFn = hash.Proto

type versionAndDataRoot struct {
	version  int
	dataRoot [32]byte
}

// AttCaches defines the caches used to satisfy attestation pool interface.
// These caches are KV store for various attestations
// such are unaggregated, aggregated or attestations within a block.
type AttCaches struct {
	aggregatedAttLock  sync.RWMutex
	aggregatedAtt      map[versionAndDataRoot][]ethpb.Att
	unAggregateAttLock sync.RWMutex
	unAggregatedAtt    map[versionAndDataRoot]ethpb.Att
	forkchoiceAttLock  sync.RWMutex
	forkchoiceAtt      map[versionAndDataRoot]ethpb.Att
	blockAttLock       sync.RWMutex
	blockAtt           map[versionAndDataRoot][]ethpb.Att
	seenAtt            *cache.Cache
}

// NewAttCaches initializes a new attestation pool consists of multiple KV store in cache for
// various kind of attestations.
func NewAttCaches() *AttCaches {
	secsInEpoch := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	c := cache.New(secsInEpoch*time.Second, 2*secsInEpoch*time.Second)
	pool := &AttCaches{
		unAggregatedAtt: make(map[versionAndDataRoot]ethpb.Att),
		aggregatedAtt:   make(map[versionAndDataRoot][]ethpb.Att),
		forkchoiceAtt:   make(map[versionAndDataRoot]ethpb.Att),
		blockAtt:        make(map[versionAndDataRoot][]ethpb.Att),
		seenAtt:         c,
	}

	return pool
}
