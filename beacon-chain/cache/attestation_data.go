package cache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

var (
	// Delay parameters
	minDelay    = float64(10)        // 10 nanoseconds
	maxDelay    = float64(100000000) // 0.1 second
	delayFactor = 1.1

	// Metrics
	attestationCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "attestation_cache_miss",
		Help: "The number of attestation data requests that aren't present in the cache.",
	})
	attestationCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "attestation_cache_hit",
		Help: "The number of attestation data requests that are present in the cache.",
	})
	attestationCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "attestation_cache_size",
		Help: "The number of attestation data in the attestations cache",
	})
)

// ErrAlreadyInProgress appears when attempting to mark a cache as in progress while it is
// already in progress. The client should handle this error and wait for the in progress
// data to resolve via Get.
var ErrAlreadyInProgress = errors.New("already in progress")

// AttestationCache is used to store the cached results of an AttestationData request.
type AttestationCache struct {
	cache      *cache.FIFO
	lock       sync.RWMutex
	inProgress map[string]bool
}

// NewAttestationCache initializes the map and underlying cache.
func NewAttestationCache() *AttestationCache {
	return &AttestationCache{
		cache:      cache.NewFIFO(wrapperToKey),
		inProgress: make(map[string]bool),
	}
}

// Get waits for any in progress calculation to complete before returning a
// cached response, if any.
func (c *AttestationCache) Get(ctx context.Context, req *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	if req == nil {
		return nil, errors.New("nil attestation data request")
	}

	s, e := reqToKey(req)
	if e != nil {
		return nil, e
	}

	delay := minDelay

	// Another identical request may be in progress already. Let's wait until
	// any in progress request resolves or our timeout is exceeded.
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		c.lock.RLock()
		if !c.inProgress[s] {
			c.lock.RUnlock()
			break
		}
		c.lock.RUnlock()

		// This increasing backoff is to decrease the CPU cycles while waiting
		// for the in progress boolean to flip to false.
		time.Sleep(time.Duration(delay) * time.Nanosecond)
		delay *= delayFactor
		delay = math.Min(delay, maxDelay)
	}

	item, exists, err := c.cache.GetByKey(s)
	if err != nil {
		return nil, err
	}

	if exists && item != nil && item.(*attestationReqResWrapper).res != nil {
		attestationCacheHit.Inc()
		return ethpb.CopyAttestationData(item.(*attestationReqResWrapper).res), nil
	}
	attestationCacheMiss.Inc()
	return nil, nil
}

// MarkInProgress a request so that any other similar requests will block on
// Get until MarkNotInProgress is called.
func (c *AttestationCache) MarkInProgress(req *ethpb.AttestationDataRequest) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	s, e := reqToKey(req)
	if e != nil {
		return e
	}
	if c.inProgress[s] {
		return ErrAlreadyInProgress
	}
	c.inProgress[s] = true
	return nil
}

// MarkNotInProgress will release the lock on a given request. This should be
// called after put.
func (c *AttestationCache) MarkNotInProgress(req *ethpb.AttestationDataRequest) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	s, e := reqToKey(req)
	if e != nil {
		return e
	}
	delete(c.inProgress, s)
	return nil
}

// Put the response in the cache.
func (c *AttestationCache) Put(_ context.Context, req *ethpb.AttestationDataRequest, res *ethpb.AttestationData) error {
	data := &attestationReqResWrapper{
		req,
		res,
	}
	if err := c.cache.AddIfNotPresent(data); err != nil {
		return err
	}
	trim(c.cache, maxCacheSize)

	attestationCacheSize.Set(float64(len(c.cache.List())))
	return nil
}

func wrapperToKey(i interface{}) (string, error) {
	w, ok := i.(*attestationReqResWrapper)
	if !ok {
		return "", errors.New("key is not of type *attestationReqResWrapper")
	}
	if w == nil {
		return "", errors.New("nil wrapper")
	}
	if w.req == nil {
		return "", errors.New("nil wrapper.request")
	}
	return reqToKey(w.req)
}

func reqToKey(req *ethpb.AttestationDataRequest) (string, error) {
	return fmt.Sprintf("%d", req.Slot), nil
}

type attestationReqResWrapper struct {
	req *ethpb.AttestationDataRequest
	res *ethpb.AttestationData
}
