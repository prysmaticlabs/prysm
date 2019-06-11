package cache

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/tools/cache"
)

var (
	// ErrNotEth1DataVote will be returned when a cache object is not a pointer to
	// a Eth1DataVote struct.
	ErrNotEth1DataVote = errors.New("object is not a eth1 data vote obj")

	// maxEth1DataVoteSize defines the max number of eth1 data votes can cache.
	maxEth1DataVoteSize = 1000

	// Metrics.
	eth1DataVoteCacheMiss = promauto.NewCounter(prometheus.CounterOpts{
		Name: "eth1_data_vote_cache_miss",
		Help: "The number of eth1 data vote count requests that aren't present in the cache.",
	})
	eth1DataVoteCacheHit = promauto.NewCounter(prometheus.CounterOpts{
		Name: "eth1_data_vote_cache_hit",
		Help: "The number of eth1 data vote count requests that are present in the cache.",
	})
)

// Eth1DataVote defines the struct which keeps track of the vote count of individual deposit root.
type Eth1DataVote struct {
	DepositRoot []byte
	VoteCount   uint64
}

// Eth1DataVoteCache is a struct with 1 queue for looking up eth1 data vote count by deposit root.
type Eth1DataVoteCache struct {
	eth1DataVoteCache *cache.FIFO
	lock              sync.RWMutex
}

// eth1DataVoteKeyFn takes the deposit root as the key for the eth1 data vote count of a given root.
func eth1DataVoteKeyFn(obj interface{}) (string, error) {
	eInfo, ok := obj.(*Eth1DataVote)
	if !ok {
		return "", ErrNotEth1DataVote
	}

	return string(eInfo.DepositRoot), nil
}

// NewEth1DataVoteCache creates a new eth1 data vote count cache for storing/accessing Eth1DataVote.
func NewEth1DataVoteCache() *Eth1DataVoteCache {
	return &Eth1DataVoteCache{
		eth1DataVoteCache: cache.NewFIFO(eth1DataVoteKeyFn),
	}
}

// Eth1DataVote fetches eth1 data vote count by deposit root. Returns vote count,
// if exists. Otherwise returns false, nil.
func (c *Eth1DataVoteCache) Eth1DataVote(depositRoot []byte) (uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.eth1DataVoteCache.GetByKey(string(depositRoot))
	if err != nil {
		return 0, err
	}

	if exists {
		eth1DataVoteCacheHit.Inc()
	} else {
		eth1DataVoteCacheMiss.Inc()
		return 0, nil
	}

	eInfo, ok := obj.(*Eth1DataVote)
	if !ok {
		return 0, ErrNotEth1DataVote
	}

	return eInfo.VoteCount, nil
}

// AddEth1DataVote adds eth1 data vote object to the cache. This method also trims the least
// recently added Eth1DataVoteByEpoch object if the cache size has ready the max cache size limit.
func (c *Eth1DataVoteCache) AddEth1DataVote(eth1DataVote *Eth1DataVote) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.eth1DataVoteCache.Add(eth1DataVote); err != nil {
		return err
	}

	trim(c.eth1DataVoteCache, maxEth1DataVoteSize)
	return nil
}

// IncrementEth1DataVote increments the existing eth1 data object's vote count by 1,
// and returns the vote count.
func (c *Eth1DataVoteCache) IncrementEth1DataVote(depositRoot []byte) (uint64, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, exists, err := c.eth1DataVoteCache.GetByKey(string(depositRoot))
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errors.New("eth1 data vote object does not exist")
	}

	eth1DataVoteCacheHit.Inc()

	eInfo, _ := obj.(*Eth1DataVote)
	eInfo.VoteCount++

	if err := c.eth1DataVoteCache.Add(eInfo); err != nil {
		return 0, err
	}

	return eInfo.VoteCount, nil
}
