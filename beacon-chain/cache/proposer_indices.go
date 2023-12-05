package cache

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

var (
	// ProposerIndicesCacheMiss tracks the number of proposerIndices requests that aren't present in the cache.
	ProposerIndicesCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proposer_indices_cache_miss",
		Help: "The number of proposer indices requests that aren't present in the cache.",
	})
	// ProposerIndicesCacheHit tracks the number of proposerIndices requests that are in the cache.
	ProposerIndicesCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "proposer_indices_cache_hit",
		Help: "The number of proposer indices requests that are present in the cache.",
	})
)

// ProposerIndicesCache keeps track of the proposer indices in the next two
// epochs. It is keyed by the state root of the last epoch before. That is, for
// blocks during epoch 2, for example slot 65, it will be keyed by the state
// root of slot 63 (last slot in epoch 1).
// The cache keeps two sets of indices computed, the "safe" set is computed
// right before the epoch transition into the current epoch. For example for
// epoch 2 we will compute this list after importing block 63. The "unsafe"
// version is computed an epoch in advance, for example for epoch 3, it will be
// computed after importing block 63.
type ProposerIndicesCache struct {
	sync.Mutex
	indices       map[primitives.Epoch]map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex
	unsafeIndices map[primitives.Epoch]map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex
}

// NewProposerIndicesCache returns a newly created cache
func NewProposerIndicesCache() *ProposerIndicesCache {
	return &ProposerIndicesCache{
		indices:       make(map[primitives.Epoch]map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex),
		unsafeIndices: make(map[primitives.Epoch]map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex),
	}
}

// ProposerIndices returns the proposer indices (safe) for the given root
func (p *ProposerIndicesCache) ProposerIndices(epoch primitives.Epoch, root [32]byte) ([fieldparams.SlotsPerEpoch]primitives.ValidatorIndex, bool) {
	p.Lock()
	defer p.Unlock()
	inner, ok := p.indices[epoch]
	if !ok {
		ProposerIndicesCacheMiss.Inc()
		return [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}, false
	}
	indices, exists := inner[root]
	if exists {
		ProposerIndicesCacheHit.Inc()
	} else {
		ProposerIndicesCacheMiss.Inc()
	}
	return indices, exists
}

// UnsafeProposerIndices returns the proposer indices (unsafe) for the given root
func (p *ProposerIndicesCache) UnsafeProposerIndices(epoch primitives.Epoch, root [32]byte) ([fieldparams.SlotsPerEpoch]primitives.ValidatorIndex, bool) {
	p.Lock()
	defer p.Unlock()
	inner, ok := p.unsafeIndices[epoch]
	if !ok {
		return [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex{}, false
	}
	indices, exists := inner[root]
	return indices, exists
}

// Prune resets the ProposerIndicesCache to its initial state
func (p *ProposerIndicesCache) Prune(epoch primitives.Epoch) {
	p.Lock()
	defer p.Unlock()
	for key := range p.indices {
		if key < epoch {
			delete(p.indices, key)
		}
	}
	for key := range p.unsafeIndices {
		if key < epoch {
			delete(p.unsafeIndices, key)
		}
	}
}

// Set sets the proposer indices for the given root as key
func (p *ProposerIndicesCache) Set(epoch primitives.Epoch, root [32]byte, indices [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex) {
	p.Lock()
	defer p.Unlock()

	inner, ok := p.indices[epoch]
	if !ok {
		inner = make(map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex)
		p.indices[epoch] = inner
	}
	inner[root] = indices
}

// Set sets the unsafe proposer indices for the given root as key
func (p *ProposerIndicesCache) SetUnsafe(epoch primitives.Epoch, root [32]byte, indices [fieldparams.SlotsPerEpoch]primitives.ValidatorIndex) {
	p.Lock()
	defer p.Unlock()
	inner, ok := p.unsafeIndices[epoch]
	if !ok {
		inner = make(map[[32]byte][fieldparams.SlotsPerEpoch]primitives.ValidatorIndex)
		p.unsafeIndices[epoch] = inner
	}
	inner[root] = indices
}
