package blockchain

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
)

type stateBalanceCache struct {
	sync.Mutex
	balances []uint64
	root     [32]byte
	stateGen balanceByRooter
}

type balanceByRooter interface {
	BalancesByRoot(context.Context, [32]byte) ([]uint64, error)
}

// newStateBalanceCache exists to remind us that stateBalanceCache needs a state gen
// to avoid nil pointer bugs when updating the cache in the read path (get())
func newStateBalanceCache(sg *stategen.State) (*stateBalanceCache, error) {
	if sg == nil {
		return nil, errors.New("can't initialize state balance cache without stategen")
	}
	return &stateBalanceCache{stateGen: sg}, nil
}

// update is called by get() when the requested root doesn't match
// the previously read value. This cache assumes we only want to cache one
// set of balances for a single root (the current justified root).
//
// WARNING: this is not thread-safe on its own, relies on get() for locking
func (c *stateBalanceCache) update(ctx context.Context, justifiedRoot [32]byte) ([]uint64, error) {
	stateBalanceCacheMiss.Inc()

	justifiedBalances, err := c.stateGen.BalancesByRoot(ctx, justifiedRoot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get balances")
	}

	c.balances = justifiedBalances
	c.root = justifiedRoot
	return c.balances, nil
}

// getBalances takes an explicit justifiedRoot so it can invalidate the singleton cache key
// when the justified root changes, and takes a context so that the long-running stategen
// read path can connect to the upstream cancellation/timeout chain.
func (c *stateBalanceCache) get(ctx context.Context, justifiedRoot [32]byte) ([]uint64, error) {
	c.Lock()
	defer c.Unlock()

	if justifiedRoot != [32]byte{} && justifiedRoot == c.root {
		stateBalanceCacheHit.Inc()
		return c.balances, nil
	}

	return c.update(ctx, justifiedRoot)
}
