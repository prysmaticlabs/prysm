package cache

import (
	"context"
	"errors"
	"sync"

	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

type AttestationConsensusData struct {
	Slot     primitives.Slot
	HeadRoot []byte
	Target   forkchoicetypes.Checkpoint
	Source   forkchoicetypes.Checkpoint
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

// Get retrieves cached attestation data, recording a cache hit or miss. This method is lock free.
func (c *AttestationCache) Get(ctx context.Context) (*AttestationConsensusData, error) {
	return c.a, nil
}

// Put adds a response to the cache. This method is lock free.
func (c *AttestationCache) Put(ctx context.Context, a *AttestationConsensusData) error {
	if a == nil {
		return errors.New("attestation cannot be nil")
	}
	c.a = a
	return nil
}

// Lock locks the cache for writing.
func (c *AttestationCache) Lock() {
	c.lock.Lock()
}

// Unlock unlocks the cache for writing.
func (c *AttestationCache) Unlock() {
	c.lock.Unlock()
}

// RLock locks the cache for reading.
func (c *AttestationCache) RLock() {
	c.lock.RLock()
}

// RUnlock unlocks the cache for reading.
func (c *AttestationCache) RUnlock() {
	c.lock.RUnlock()
}
