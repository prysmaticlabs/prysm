package cache

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
)

// generics
type Key = primitives.Epoch
type Value = []byte

var (
	// case setup
	myCache     lruCache[Key, Value]
	myCacheOnce sync.Once
	cacheSetup  = func(t *testing.T) {
		myCacheOnce.Do(func() {
			var err error
			myCache, err = NewTestCache[Key, Value]()
			if err != nil {
				t.Fatalf("Error creating cache: %v", err)
			}
		})
		myCache.Clear()
	}

	// case metrics
	reg = prometheus.NewPedanticRegistry()

	// case values
	key   = primitives.Epoch(1)
	value = []byte("0xaaa")
)

func TestCache_LRU(t *testing.T) {
	tests := []struct {
		name                     string
		cacheSetup               func(t *testing.T)
		key                      Key
		value                    Value
		expectedValue            Value
		expectedError            error
		expectedHitCacheMetrics  float64
		expectedMissCacheMetrics float64
	}{
		{
			name: "Test adding value returns value",
			cacheSetup: func(t *testing.T) {
				cacheSetup(t)
			},
			key:                      key,
			value:                    value,
			expectedValue:            value,
			expectedError:            nil,
			expectedHitCacheMetrics:  1,
			expectedMissCacheMetrics: 0,
		},
		{
			name: "Test adding nil value returns error",
			cacheSetup: func(t *testing.T) {
				cacheSetup(t)
			},
			key:                      key,
			value:                    nil,
			expectedValue:            nil,
			expectedError:            ErrNilValueProvided,
			expectedHitCacheMetrics:  1,
			expectedMissCacheMetrics: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// test setup
			test.cacheSetup(t)

			// test values
			err := add(myCache, test.key, test.value)
			if !errors.Is(err, test.expectedError) {
				t.Errorf("Expected error %v, got %v", test.expectedError, err)
			}

			var item Value
			item, err = get(myCache, test.key)
			if item == nil {
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("Expected error %v, got %v", ErrNotFound, err)
				}
			}
			require.DeepEqual(t, item, test.expectedValue)

			// test metrics
			var metrics []*dto.MetricFamily
			if metrics, err = reg.Gather(); err != nil {
				t.Error("Gathering failed:", err)
			}
			assert.Equal(t, dto.MetricType_COUNTER, metrics[0].GetType())
			assert.Equal(t, "total_test_cache_hit", metrics[0].GetName())
			assert.Equal(t, "The number of get requests that are present in the cache.", metrics[0].GetHelp())
			assert.Equal(t, test.expectedHitCacheMetrics, *metrics[0].GetMetric()[0].Counter.Value)

			assert.Equal(t, dto.MetricType_COUNTER, metrics[1].GetType())
			assert.Equal(t, "total_test_cache_miss", metrics[1].GetName())
			assert.Equal(t, "The number of get requests that aren't present in the cache.", metrics[1].GetHelp())
			assert.Equal(t, test.expectedMissCacheMetrics, *metrics[1].GetMetric()[0].Counter.Value)
		})
	}
}

const (
	maxTestCacheSize = int(4)
)

var (
	testPromCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_test_cache_hit",
		Help: "The number of get requests that are present in the cache.",
	})
	testPromCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "total_test_cache_miss",
		Help: "The number of get requests that aren't present in the cache.",
	})
)

type TestCache[K Key, V Value] struct {
	lru                         *lru.Cache[K, V]
	promCacheMiss, promCacheHit prometheus.Counter
}

func NewTestCache[K Key, V Value]() (*TestCache[K, V], error) {
	cache, err := lru.New[K, V](maxTestCacheSize)
	if err != nil {
		return nil, err
	}

	if testPromCacheHit == nil || testPromCacheMiss == nil {
		return nil, err
	}

	reg.MustRegister(testPromCacheMiss)
	reg.MustRegister(testPromCacheHit)

	return &TestCache[K, V]{
		lru:           cache,
		promCacheMiss: testPromCacheMiss,
		promCacheHit:  testPromCacheHit,
	}, nil
}

func (c *TestCache[K, V]) get() *lru.Cache[K, V] {
	return c.lru
}

func (c *TestCache[K, V]) hitCache() {
	c.promCacheHit.Inc()
}

func (c *TestCache[K, V]) missCache() {
	c.promCacheMiss.Inc()
}

func (c *TestCache[K, V]) Clear() {
	purge[K, V](c)
}
