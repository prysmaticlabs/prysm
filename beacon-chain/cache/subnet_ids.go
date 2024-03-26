package cache

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/container/slice"
)

var (
	// subnetIDsCacheMiss tracks the number of subnet ids requests that aren't present in the cache.
	subnetIDsAttesterCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_subnet_ids_attester_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
	// subnetIDsCacheHit tracks the number of subnet ids requests that are in the cache.
	subnetIDsAttesterCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_subnet_ids_attester_cache_hit",
		Help: "The number of get requests that are present in the cache.",
	})
)

type subnetIDsAttesterCache[K primitives.Slot, V []uint64] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter
}

func newSubnetIDsAttesterCache[K primitives.Slot, V []uint64](cacheSize int) (*subnetIDsAttesterCache[K, V], error) {
	attesterLRUCache, err := lru.New[K, V](cacheSize)
	if err != nil {
		return nil, errors.Wrap(ErrCacheCannotBeNil, "attester cache initialisation failed")
	}

	if subnetIDsAttesterCacheHit == nil || subnetIDsAttesterCacheMiss == nil {
		return nil, ErrCacheMetricsCannotBeNil
	}

	return &subnetIDsAttesterCache[K, V]{
		lru:           attesterLRUCache,
		promCacheMiss: subnetIDsAttesterCacheMiss,
		promCacheHit:  subnetIDsAttesterCacheHit,
	}, nil
}

func (s *subnetIDsAttesterCache[K, V]) get() *lru.Cache[K, V] { //nolint: unused, -- bug in golangci-lint 1.55
	return s.lru
}

func (s *subnetIDsAttesterCache[K, V]) hitCache() { //nolint: unused, -- bug in golangci-lint 1.55
	s.promCacheHit.Inc()
}

func (s *subnetIDsAttesterCache[K, V]) missCache() { //nolint: unused, -- bug in golangci-lint 1.55
	s.promCacheMiss.Inc()
}

// Clear the BalanceCache to its initial state
func (s *subnetIDsAttesterCache[K, V]) Clear() { //nolint: unused, -- bug in golangci-lint 1.55
	purge[K, V](s)
}

// ---------------------------------------------------------------------------------------------------------------------

var (
	// subnetIDsCacheMiss tracks the number of subnet ids requests that aren't present in the cache.
	subnetIDsAggregatorCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_subnet_ids_aggregator_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
	// subnetIDsCacheHit tracks the number of subnet ids requests that are in the cache.
	subnetIDsAggregatorCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_subnet_ids_aggregator_cache_hit",
		Help: "The number of get requests that are present in the cache.",
	})
)

type subnetIDsAggregatorCache[K primitives.Slot, V []uint64] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter
}

func newSubnetIDsAggregatorCache[K primitives.Slot, V []uint64](cacheSize int) (*subnetIDsAggregatorCache[K, V], error) {
	cache, err := lru.New[K, V](cacheSize)
	if err != nil {
		return nil, ErrCacheCannotBeNil
	}

	if subnetIDsAttesterCacheHit == nil || subnetIDsAttesterCacheMiss == nil {
		return nil, ErrCacheMetricsCannotBeNil
	}

	return &subnetIDsAggregatorCache[K, V]{
		lru:           cache,
		promCacheMiss: subnetIDsAggregatorCacheMiss,
		promCacheHit:  subnetIDsAggregatorCacheHit,
	}, nil
}

func (s *subnetIDsAggregatorCache[K, V]) get() *lru.Cache[K, V] { //nolint: unused, -- bug in golangci-lint 1.55
	return s.lru
}

func (s *subnetIDsAggregatorCache[K, V]) hitCache() { //nolint: unused, -- bug in golangci-lint 1.55
	s.promCacheHit.Inc()
}

func (s *subnetIDsAggregatorCache[K, V]) missCache() { //nolint: unused, -- bug in golangci-lint 1.55
	s.promCacheMiss.Inc()
}

// Clear the BalanceCache to its initial state
func (s *subnetIDsAggregatorCache[K, V]) Clear() { //nolint: unused, -- bug in golangci-lint 1.55
	purge[K, V](s)
}

// ---------------------------------------------------------------------------------------------------------------------

// SubnetIDs for attester and aggregator.
var SubnetIDs, _ = newSubnetIDs()

var subnetKey = "persistent-subnets"

type subnetIDs[K primitives.Slot, V []uint64] struct {
	m sync.RWMutex

	attesterCache     Cache[K, V]
	aggregatorCache   Cache[K, V]
	persistentSubnets *cache.Cache
}

// NewEffectiveBalanceCache creates a new effective balance cache for storing/accessing total balance by epoch.
func newSubnetIDs[K primitives.Slot, V []uint64]() (*subnetIDs[K, V], error) {
	// Given a node can calculate committee assignments of current epoch and next epoch.
	// Max size is set to 2 epoch length.
	cacheSize := int(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxCommitteesPerSlot * 2)) // lint:ignore uintcast -- constant values that would panic on startup if negative.

	attesterLRUCache, err := newSubnetIDsAttesterCache[K, V](cacheSize)
	if err != nil {
		return nil, errors.Wrap(err, "attester cache initialisation failed")
	}

	aggregatorLRUCache, err := newSubnetIDsAggregatorCache[K, V](cacheSize)
	if err != nil {
		return nil, errors.Wrap(err, "aggregator cache initialisation failed")
	}

	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	subLength := epochDuration * time.Duration(params.BeaconConfig().EpochsPerRandomSubnetSubscription)
	persistentCache := cache.New(subLength*time.Second, epochDuration*time.Second)

	return &subnetIDs[K, V]{
		attesterCache:     attesterLRUCache,
		aggregatorCache:   aggregatorLRUCache,
		persistentSubnets: persistentCache,
	}, nil
}

// AddAttesterSubnetID adds the subnet index for subscribing subnet for the attester of a given slot.
func (s *subnetIDs[K, V]) AddAttesterSubnetID(slot K, subnetID uint64) error {
	s.m.Lock()
	defer s.m.Unlock()

	ids := V{subnetID}
	items, err := get(s.attesterCache, slot)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	ids = slice.UnionUint64(items, ids)
	return add(s.attesterCache, slot, ids)
}

// GetAttesterSubnetIDs gets the subnet IDs for subscribed subnets for attesters of the slot.
func (s *subnetIDs[K, V]) GetAttesterSubnetIDs(slot K) (V, error) {
	items, err := get(s.attesterCache, slot)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return items, nil
}

// AddAggregatorSubnetID adds the subnet ID for subscribing subnet for the aggregator of a given slot.
func (s *subnetIDs[K, V]) AddAggregatorSubnetID(slot K, subnetID uint64) error {
	s.m.Lock()
	defer s.m.Unlock()

	ids := V{subnetID}
	items, err := get(s.aggregatorCache, slot)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	ids = slice.UnionUint64(items, ids)
	return add(s.aggregatorCache, slot, ids)
}

// GetAggregatorSubnetIDs gets the subnet IDs for subscribing subnet for aggregator of the slot.
func (s *subnetIDs[K, V]) GetAggregatorSubnetIDs(slot K) (V, error) {
	items, err := get(s.aggregatorCache, slot)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
	}
	return items, nil
}

// GetPersistentSubnets retrieves the persistent subnet and expiration time of that validator's
// subscription.
func (s *subnetIDs[K, V]) GetPersistentSubnets() (V, bool, time.Time) {
	id, duration, ok := s.persistentSubnets.GetWithExpiration(subnetKey)
	if !ok {
		return V{}, ok, time.Time{}
	}
	return id.(V), ok, duration
}

// GetAllSubnets retrieves all the non-expired subscribed subnets of all the validators
// in the cache.
func (s *subnetIDs[K, V]) GetAllSubnets() V {
	var committees V
	itemsMap := s.persistentSubnets.Items()
	for _, v := range itemsMap {
		if v.Expired() {
			continue
		}
		committees = append(committees, v.Object.(V)...)
	}
	return slice.SetUint64(committees)
}

// AddPersistentCommittee adds the relevant committee for that particular validator along with its
// expiration period.
func (s *subnetIDs[K, V]) AddPersistentCommittee(comIndex V, duration time.Duration) {
	s.persistentSubnets.Set(subnetKey, comIndex, duration)
}

// EmptyAllCaches empties out all the related caches and flushes any stored
// entries on them. This should only ever be used for testing, in normal
// production, handling of the relevant subnets for each role is done
// separately.
func (s *subnetIDs[K, V]) EmptyAllCaches() {
	purge[K, V](s.attesterCache)
	purge[K, V](s.aggregatorCache)
	s.persistentSubnets.Flush()
}
