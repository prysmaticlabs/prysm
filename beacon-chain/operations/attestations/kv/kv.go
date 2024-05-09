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
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
)

var hashFn = hash.Proto

type AttestationId struct {
	version int
	digest  [32]byte
}

// TODO doesn't make sense to implement like this
func NewAttestationId(att ethpb.Att, digest [32]byte) AttestationId {
	if att.Version() == version.Phase0 {
		return AttestationId{
			version: att.Version(),
			digest:  digest,
		}
	}
	return AttestationId{
		version: att.Version(),
		digest:  digest,
	}
}

// AttCaches defines the caches used to satisfy attestation pool interface.
// These caches are KV store for various attestations
// such are unaggregated, aggregated or attestations within a block.
type AttCaches struct {
	aggregatedAttLock  sync.RWMutex
	aggregatedAtt      map[AttestationId][]ethpb.Att
	unAggregateAttLock sync.RWMutex
	unAggregatedAtt    map[AttestationId]ethpb.Att
	forkchoiceAttLock  sync.RWMutex
	forkchoiceAtt      map[AttestationId]ethpb.Att
	blockAttLock       sync.RWMutex
	blockAtt           map[AttestationId][]ethpb.Att
	seenAtt            *cache.Cache
}

// NewAttCaches initializes a new attestation pool consists of multiple KV store in cache for
// various kind of attestations.
func NewAttCaches() *AttCaches {
	secsInEpoch := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	c := cache.New(secsInEpoch*time.Second, 2*secsInEpoch*time.Second)
	pool := &AttCaches{
		unAggregatedAtt: make(map[AttestationId]ethpb.Att),
		aggregatedAtt:   make(map[AttestationId][]ethpb.Att),
		forkchoiceAtt:   make(map[AttestationId]ethpb.Att),
		blockAtt:        make(map[AttestationId][]ethpb.Att),
		seenAtt:         c,
	}

	return pool
}
