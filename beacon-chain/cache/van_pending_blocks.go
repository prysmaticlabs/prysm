package cache

import (
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"strconv"
	"sync"
)

var (
	// maxPendingBlocksCacheSize defines the max number of pending blocks can cache for waiting orchestrator confirmation.
	// beacon block size is about 2KB so, 1000 * 2KB = 2MB can be cached. It should be enough because each pending block
	// should be finalized within this window.
	maxPendingBlocksCacheSize = uint64(1000)

	// ErrNotBeaconBlock will be returned when a cache object is not a pointer to
	// a beacon block type.
	ErrNotBeaconBlock = errors.New("object is not a beacon block type")
)

// PendingBlocksCache is a struct with 1 queue for looking up pending blocks.
type PendingBlocksCache struct {
	PendingBlocksCache *cache.FIFO
	lock               sync.RWMutex
}

// pendingBlocksKeyFn takes the slot number as the key to retrieve pending blocks from queue.
func pendingBlocksKeyFn(obj interface{}) (string, error) {
	block, ok := obj.(*ethpb.BeaconBlock)
	if !ok {
		return "", ErrNotBeaconBlock
	}
	return slotToString(block.GetSlot()), nil
}

// NewCommitteesCache creates a new committee cache for storing/accessing shuffled indices of a committee.
func NewPendingBlocksCache() *PendingBlocksCache {
	return &PendingBlocksCache{
		PendingBlocksCache: cache.NewFIFO(pendingBlocksKeyFn),
	}
}

// AddPendingBlock adds pending beacon block object to the cache.
// This method also trims the least recently list if the cache size has ready the max cache size limit.
func (c *PendingBlocksCache) AddPendingBlock(blk *ethpb.BeaconBlock) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.PendingBlocksCache.Add(blk); err != nil {
		return err
	}
	trim(c.PendingBlocksCache, maxPendingBlocksCacheSize)
	return nil
}

// PendingBlock returns the pending block of the given slot number
func (c *PendingBlocksCache) PendingBlock(slot types.Slot) (*ethpb.BeaconBlock, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists, err := c.PendingBlocksCache.GetByKey(slotToString(slot))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	item, ok := obj.(*ethpb.BeaconBlock)
	if !ok {
		return nil, ErrNotBeaconBlock
	}

	return item, nil
}

// ProposerIndices returns all the pending blocks
func (c *PendingBlocksCache) PendingBlocks() ([]*ethpb.BeaconBlock, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := c.PendingBlocksCache.ListKeys()
	pendingBlocks := make([]*ethpb.BeaconBlock, len(keys))
	for i, key := range keys {
		obj, exists, err := c.PendingBlocksCache.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, nil
		}

		item, ok := obj.(*ethpb.BeaconBlock)
		if !ok {
			return nil, ErrNotBeaconBlock
		}
		//pendingBlocks = append(pendingBlocks, item)
		pendingBlocks[i] = item
	}

	return pendingBlocks, nil
}

// Delete deletes the confirmed block from cache
func (c *PendingBlocksCache) Delete(slot types.Slot) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, exists, err := c.PendingBlocksCache.GetByKey(slotToString(slot))
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	item, ok := obj.(*ethpb.BeaconBlock)
	if !ok {
		return ErrNotBeaconBlock
	}

	return c.PendingBlocksCache.Delete(item)
}

// Converts input uint64 to string. To be used as key for slot to get root.
func slotToString(s types.Slot) string {
	return strconv.FormatUint(uint64(s), 10)
}
