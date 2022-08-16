package blockchain

import (
	"context"
	"errors"
	"testing"

	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
)

func testServiceOptsWithDB(t *testing.T) []Option {
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New()
	return []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
}

// WARNING: only use these opts when you are certain there are no db calls
// in your code path. this is a lightweight way to satisfy the stategen/beacondb
// initialization requirements w/o the overhead of db init.
func testServiceOptsNoDB() []Option {
	return []Option{
		withStateBalanceCache(satisfactoryStateBalanceCache()),
	}
}

type mockStateByRooter struct {
	state state.BeaconState
	err   error
}

var _ stateByRooter = &mockStateByRooter{}

func (m mockStateByRooter) StateByRoot(_ context.Context, _ [32]byte) (state.BeaconState, error) {
	return m.state, m.err
}

// returns an instance of the state balance cache that can be used
// to satisfy the requirement for one in NewService, but which will
// always return an error if used.
func satisfactoryStateBalanceCache() *stateBalanceCache {
	err := errors.New("satisfactoryStateBalanceCache doesn't perform real caching")
	return &stateBalanceCache{stateGen: mockStateByRooter{err: err}}
}
