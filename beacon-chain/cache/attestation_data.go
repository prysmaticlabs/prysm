package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type AttestationConsensusData struct {
	Slot             primitives.Slot
	HeadRoot         []byte
	TargetCheckpoint *ethpb.Checkpoint
	SourceCheckpoint *ethpb.Checkpoint
}

// AttestationCache stores cached results of AttestationData requests.
type AttestationCache struct {
	a    *AttestationConsensusData
	lock sync.RWMutex
}

// NewAttestationCache creates a new instance of AttestationCache.
func NewAttestationCache() *AttestationCache {
	return &AttestationCache{}
}

// Get retrieves cached attestation data, recording a cache hit or miss.
func (c *AttestationCache) Get(ctx context.Context) (*AttestationConsensusData, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.a, nil
}

// Put adds a response to the cache.
func (c *AttestationCache) Put(ctx context.Context, a *AttestationConsensusData) error {
	if a == nil {
		return errors.New("attestation cannot be nil")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	c.a = a

	return nil
}
