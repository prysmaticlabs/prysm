package stategen

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
)

var ErrNotInCache = errors.New("state not found in cache")

type CachedGetter interface {
	ByRoot([32]byte) (state.BeaconState, error)
}

type CombinedCache struct {
	getters []CachedGetter
}

func (c CombinedCache) ByRoot(root [32]byte) (state.BeaconState, error) {
	for _, getter := range c.getters {
		st, err := getter.ByRoot(root)
		if err == nil {
			return st, nil
		}
		if errors.Is(err, ErrNotInCache) {
			continue
		}
		return nil, err
	}
	return nil, ErrNotInCache
}

var _ CachedGetter = &CombinedCache{}
