package blockchain

import (
	"context"
	"errors"
	"sync"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
)

type stateBalanceCache struct {
	sync.RWMutex
	balances []uint64
	root [32]byte
	stateGen                *stategen.State
}

// stateBalanceCache wants a stagegen for updating the cache in the readpath,
// so NewStateBalanceCache exists to remind us it needs that bit of special setup.
func NewStateBalanceCache(sg *stategen.State) *stateBalanceCache {
	return &stateBalanceCache{stateGen: sg}
}

// updateCache is usually called by getBalances when the requested root doesn't match
// the previously read value. This cache assumes only want to cache one set of balances
// for a single root (the current justified root).
func (c *stateBalanceCache) update(ctx context.Context, justifiedRoot [32]byte) error {
	justifiedState, err := c.stateGen.StateByRoot(ctx, justifiedRoot)
	if err != nil {
		return err
	}
	if justifiedState == nil || justifiedState.IsNil() {
		return errors.New("justified state can't be nil")
	}
	epoch := time.CurrentEpoch(justifiedState)

	justifiedBalances := make([]uint64, justifiedState.NumValidators())
	var balanceAccumulator = func(idx int, val state.ReadOnlyValidator) error {
		if helpers.IsActiveValidatorUsingTrie(val, epoch) {
			justifiedBalances[idx] = val.EffectiveBalance()
		} else {
			justifiedBalances[idx] = 0
		}
		return nil
	}
	if err := justifiedState.ReadFromEveryValidator(balanceAccumulator); err != nil {
		return err
	}

	// TODO considering the nature of this cache, should the whole method be in the critical section?
	c.Lock()
	defer c.Unlock()
	c.balances = justifiedBalances
	return nil
}

// getBalances takes an explicit justifiedRoot so it can invalidate the singleton cache key
// when the justified root changes, and takes a context so that the long-running stategen
// read path can connect to the upstream cancellation/timeout chain.
func (c *stateBalanceCache) get(ctx context.Context, justifiedRoot [32]byte) ([]uint64, error) {
	// justified root has changed since last read, update
	if justifiedRoot != c.root {
		if err := c.update(ctx, justifiedRoot); err != nil {
			return nil, err
		}
	}

	c.RLock()
	defer c.RUnlock()
	return c.balances, nil
}
