package cache

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type subnetIDs struct {
	attester          *lru.Cache
	attesterLock      sync.RWMutex
	aggregator        *lru.Cache
	aggregatorLock    sync.RWMutex
	persistentSubnets *cache.Cache
	subnetsLock       sync.RWMutex
}

// SubnetIDs for attester and aggregator.
var SubnetIDs = newSubnetIDs()

func newSubnetIDs() *subnetIDs {
	// Given a node can calculate committee assignments of current epoch and next epoch.
	// Max size is set to 2 epoch length.
	cacheSize := params.BeaconConfig().MaxCommitteesPerSlot * params.BeaconConfig().SlotsPerEpoch * 2
	attesterCache, err := lru.New(int(cacheSize))
	if err != nil {
		panic(err)
	}
	aggregatorCache, err := lru.New(int(cacheSize))
	if err != nil {
		panic(err)
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	subLength := epochDuration * time.Duration(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription)
	persistentCache := cache.New(subLength*time.Second, epochDuration*time.Second)
	return &subnetIDs{attester: attesterCache, aggregator: aggregatorCache, persistentSubnets: persistentCache}
}

// AddAttesterSubnetID adds the subnet index for subscribing subnet for the attester of a given slot.
func (c *subnetIDs) AddAttesterSubnetID(slot uint64, subnetID uint64) {
	c.attesterLock.Lock()
	defer c.attesterLock.Unlock()

	ids := []uint64{subnetID}
	val, exists := c.attester.Get(slot)
	if exists {
		ids = sliceutil.UnionUint64(append(val.([]uint64), ids...))
	}
	c.attester.Add(slot, ids)
}

// GetAttesterSubnetIDs gets the subnet IDs for subscribed subnets for attesters of the slot.
func (c *subnetIDs) GetAttesterSubnetIDs(slot uint64) []uint64 {
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

// AddAggregatorSubnetID adds the subnet ID for subscribing subnet for the aggregator of a given slot.
func (c *subnetIDs) AddAggregatorSubnetID(slot uint64, subnetID uint64) {
	c.aggregatorLock.Lock()
	defer c.aggregatorLock.Unlock()

	ids := []uint64{subnetID}
	val, exists := c.aggregator.Get(slot)
	if exists {
		ids = sliceutil.UnionUint64(append(val.([]uint64), ids...))
	}
	c.aggregator.Add(slot, ids)
}

// GetAggregatorSubnetIDs gets the subnet IDs for subscribing subnet for aggregator of the slot.
func (c *subnetIDs) GetAggregatorSubnetIDs(slot uint64) []uint64 {
	c.aggregatorLock.RLock()
	defer c.aggregatorLock.RUnlock()

	val, exists := c.aggregator.Get(slot)
	if !exists {
		return []uint64{}
	}
	return val.([]uint64)
}

// GetPersistentSubnets retrieves the persistent subnet and expiration time of that validator's
// subscription.
func (c *subnetIDs) GetPersistentSubnets(pubkey []byte) ([]uint64, bool, time.Time) {
	c.subnetsLock.RLock()
	defer c.subnetsLock.RUnlock()

	id, duration, ok := c.persistentSubnets.GetWithExpiration(string(pubkey))
	if !ok {
		return []uint64{}, ok, time.Time{}
	}
	return id.([]uint64), ok, duration
}

// GetAllSubnets retrieves all the non-expired subscribed subnets of all the validators
// in the cache.
func (c *subnetIDs) GetAllSubnets() []uint64 {
	c.subnetsLock.RLock()
	defer c.subnetsLock.RUnlock()

	itemsMap := c.persistentSubnets.Items()
	committees := []uint64{}

	for _, v := range itemsMap {
		if v.Expired() {
			continue
		}
		committees = append(committees, v.Object.([]uint64)...)
	}
	return sliceutil.SetUint64(committees)
}

// AddPersistentCommittee adds the relevant committee for that particular validator along with its
// expiration period.
func (c *subnetIDs) AddPersistentCommittee(pubkey []byte, comIndex []uint64, duration time.Duration) {
	c.subnetsLock.Lock()
	defer c.subnetsLock.Unlock()

	c.persistentSubnets.Set(string(pubkey), comIndex, duration)
}
