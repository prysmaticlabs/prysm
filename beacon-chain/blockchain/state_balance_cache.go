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

func NewStateBalanceCache(sg *stategen.State) *stateBalanceCache {
	return &stateBalanceCache{stateGen: sg}
}

func (c *stateBalanceCache) updateCache(ctx context.Context, justifiedRoot [32]byte) error {
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

	c.Lock()
	defer c.Unlock()
	c.balances = justifiedBalances
	return nil
}

func (c *stateBalanceCache) getBalances(ctx context.Context, justifiedRoot [32]byte) ([]uint64, error) {
	// justified root has changed since last read, update
	if justifiedRoot != c.root {
		if err := c.updateCache(ctx, justifiedRoot); err != nil {
			return nil, err
		}
	}
	c.RLock()
	defer c.RUnlock()
	return c.balances, nil
}
