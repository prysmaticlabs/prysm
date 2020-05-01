package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"

	lru "github.com/hashicorp/golang-lru"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type committeeIDs struct {
	attester          *lru.Cache
	attesterLock      sync.RWMutex
	aggregator        *lru.Cache
	aggregatorLock    sync.RWMutex
	persistentSubnets *cache.Cache
}

// CommitteeIDs for attester and aggregator.
var CommitteeIDs = newCommitteeIDs()

func newCommitteeIDs() *committeeIDs {
	// Given a node can calculate committee assignments of current epoch and next epoch.
	// Max size is set to 2 epoch length.
	cacheSize := int(params.BeaconConfig().MaxCommitteesPerSlot * params.BeaconConfig().SlotsPerEpoch * 2)
	attesterCache, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}
	aggregatorCache, err := lru.New(cacheSize)
	if err != nil {
		panic(err)
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	subLength := epochDuration * time.Duration(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription)
	persistentCache := cache.New(subLength*time.Second, epochDuration*time.Second)
	return &committeeIDs{attester: attesterCache, aggregator: aggregatorCache, persistentSubnets: persistentCache}
}

// AddAttesterCommiteeID adds committee ID for subscribing subnet for the attester of a given slot.
func (c *committeeIDs) AddAttesterCommiteeID(slot uint64, committeeID uint64) {
	c.attesterLock.Lock()
	defer c.attesterLock.Unlock()

	ids := []uint64{committeeID}
	val, exists := c.attester.Get(slot)
	if exists {
		ids = sliceutil.UnionUint64(append(val.([]uint64), ids...))
	}
	c.attester.Add(slot, ids)
}

// GetAttesterCommitteeIDs gets the committee ID for subscribing subnet for attester of the slot.
func (c *committeeIDs) GetAttesterCommitteeIDs(slot uint64) []uint64 {
	c.attesterLock.RLock()
	defer c.attesterLock.RUnlock()

	val, exists := c.attester.Get(slot)
	if !exists {
		return nil
	}
	if v, ok := val.([]uint64); ok {
		return v
	}
	return nil
}

// AddAggregatorCommiteeID adds committee ID for subscribing subnet for the aggregator of a given slot.
func (c *committeeIDs) AddAggregatorCommiteeID(slot uint64, committeeID uint64) {
	c.aggregatorLock.Lock()
	defer c.aggregatorLock.Unlock()

	ids := []uint64{committeeID}
	val, exists := c.aggregator.Get(slot)
	if exists {
		ids = sliceutil.UnionUint64(append(val.([]uint64), ids...))
	}
	c.aggregator.Add(slot, ids)
}

// GetAggregatorCommitteeIDs gets the committee ID for subscribing subnet for aggregator of the slot.
func (c *committeeIDs) GetAggregatorCommitteeIDs(slot uint64) []uint64 {
	c.aggregatorLock.RLock()
	defer c.aggregatorLock.RUnlock()

	val, exists := c.aggregator.Get(slot)
	if !exists {
		return []uint64{}
	}
	return val.([]uint64)
}

func (c *committeeIDs) AddPersistentCommittee()
