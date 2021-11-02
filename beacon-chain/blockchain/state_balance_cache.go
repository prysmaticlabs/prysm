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

var errNilStateFromStategen = errors.New("justified state can't be nil")

type stateBalanceCache struct {
	sync.Mutex
	balances	[]uint64
	root		[32]byte
	stateGen	stateByRooter
}

type stateByRooter interface {
	StateByRoot(context.Context, [32]byte) (state.BeaconState, error)
}

// stateBalanceCache wants a stagegen for updating the cache in the readpath,
// so NewStateBalanceCache exists to remind us it needs that bit of special setup.
func NewStateBalanceCache(sg *stategen.State) *stateBalanceCache {
	return &stateBalanceCache{stateGen: sg}
}

// update is called by get() when the requested root doesn't match
// the previously read value. This cache assumes we only want to cache one
// set of balances for a single root (the current justified root).
//
// warning: this is not thread-safe on its own, relies on get() for locking
func (c *stateBalanceCache) update(ctx context.Context, justifiedRoot [32]byte) ([]uint64, error) {
	stateBalanceCacheMiss.Inc()
	justifiedState, err := c.stateGen.StateByRoot(ctx, justifiedRoot)
	if err != nil {
		return nil, err
	}
	if justifiedState == nil || justifiedState.IsNil() {
		return nil, errNilStateFromStategen
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
		return nil, err
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
	if justifiedRoot == c.root {
		stateBalanceCacheHit.Inc()
		return c.balances, nil
	}

	return c.update(ctx, justifiedRoot)
}
