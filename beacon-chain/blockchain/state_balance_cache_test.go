package blockchain

import (
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	v2 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v2"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

type testStateOpt func(*ethpb.BeaconStateAltair)

func testStateWithValidators(v []*ethpb.Validator) testStateOpt {
	return func(a *ethpb.BeaconStateAltair) {
		a.Validators = v
	}
}

func testStateWithSlot(slot types.Slot) testStateOpt {
	return func(a *ethpb.BeaconStateAltair) {
		a.Slot = slot
	}
}

func testStateFixture(opts ...testStateOpt) state.BeaconState {
	a := &ethpb.BeaconStateAltair{}
	for _, o := range opts {
		o(a)
	}
	s, _ := v2.InitializeFromProtoUnsafe(a)
	return s
}

func generateTestValidators(count int, opts ...func(*ethpb.Validator)) []*ethpb.Validator {
	vs := make([]*ethpb.Validator, count)
	var i uint32 = 0
	for ; i < uint32(count); i++ {
		pk := make([]byte, 48)
		binary.LittleEndian.PutUint32(pk, i)
		v := &ethpb.Validator{PublicKey: pk}
		for _, o := range opts {
			o(v)
		}
		vs[i] = v
	}
	return vs
}

func oddValidatorsExpired(currentSlot types.Slot) func(*ethpb.Validator) {
	return func(v *ethpb.Validator) {
		pki := binary.LittleEndian.Uint64(v.PublicKey)
		if pki%2 == 0 {
			v.ExitEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) + 1)
		} else {
			v.ExitEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) - 1)
		}
	}
}

func oddValidatorsQueued(currentSlot types.Slot) func(*ethpb.Validator) {
	return func(v *ethpb.Validator) {
		v.ExitEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) + 1)
		pki := binary.LittleEndian.Uint64(v.PublicKey)
		if pki%2 == 0 {
			v.ActivationEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) - 1)
		} else {
			v.ActivationEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) + 1)
		}
	}
}

func allValidatorsValid(currentSlot types.Slot) func(*ethpb.Validator) {
	return func(v *ethpb.Validator) {
		v.ActivationEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) - 1)
		v.ExitEpoch = types.Epoch(int(slots.ToEpoch(currentSlot)) + 1)
	}
}

func balanceIsKeyTimes2(v *ethpb.Validator) {
	pki := binary.LittleEndian.Uint64(v.PublicKey)
	v.EffectiveBalance = uint64(pki) * 2
}

func testHalfExpiredValidators() ([]*ethpb.Validator, []uint64) {
	balances := []uint64{0, 0, 4, 0, 8, 0, 12, 0, 16, 0}
	return generateTestValidators(10,
		oddValidatorsExpired(types.Slot(99)),
		balanceIsKeyTimes2), balances
}

func testHalfQueuedValidators() ([]*ethpb.Validator, []uint64) {
	balances := []uint64{0, 0, 4, 0, 8, 0, 12, 0, 16, 0}
	return generateTestValidators(10,
		oddValidatorsQueued(types.Slot(99)),
		balanceIsKeyTimes2), balances
}

func testAllValidValidators() ([]*ethpb.Validator, []uint64) {
	balances := []uint64{0, 2, 4, 6, 8, 10, 12, 14, 16, 18}
	return generateTestValidators(10,
		allValidatorsValid(types.Slot(99)),
		balanceIsKeyTimes2), balances
}

func TestStateBalanceCache(t *testing.T) {
	type sbcTestCase struct {
		err      error
		root     [32]byte
		sbc      *stateBalanceCache
		balances []uint64
		name     string
	}
	sentinelCacheMiss := errors.New("cache missed, as expected")
	sentinelBalances := []uint64{1, 2, 3, 4, 5}
	halfExpiredValidators, halfExpiredBalances := testHalfExpiredValidators()
	halfQueuedValidators, halfQueuedBalances := testHalfQueuedValidators()
	allValidValidators, allValidBalances := testAllValidValidators()
	cases := []sbcTestCase{
		{
			root:     bytesutil.ToBytes32([]byte{'A'}),
			balances: sentinelBalances,
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					err: sentinelCacheMiss,
				},
				root:     bytesutil.ToBytes32([]byte{'A'}),
				balances: sentinelBalances,
			},
			name: "cache hit",
		},
		// this works by using a staterooter that returns a known error
		// so really we're testing the miss by making sure stategen got called
		// this also tells us stategen errors are propagated
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					err: sentinelCacheMiss,
				},
				root: bytesutil.ToBytes32([]byte{'B'}),
			},
			err:  sentinelCacheMiss,
			root: bytesutil.ToBytes32([]byte{'A'}),
			name: "cache miss",
		},
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{},
				root:     bytesutil.ToBytes32([]byte{'B'}),
			},
			err:  errNilStateFromStategen,
			root: bytesutil.ToBytes32([]byte{'A'}),
			name: "error for nil state upon cache miss",
		},
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					state: testStateFixture(
						testStateWithSlot(99),
						testStateWithValidators(halfExpiredValidators)),
				},
			},
			balances: halfExpiredBalances,
			root:     bytesutil.ToBytes32([]byte{'A'}),
			name:     "test filtering by exit epoch",
		},
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					state: testStateFixture(
						testStateWithSlot(99),
						testStateWithValidators(halfQueuedValidators)),
				},
			},
			balances: halfQueuedBalances,
			root:     bytesutil.ToBytes32([]byte{'A'}),
			name:     "test filtering by activation epoch",
		},
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					state: testStateFixture(
						testStateWithSlot(99),
						testStateWithValidators(allValidValidators)),
				},
			},
			balances: allValidBalances,
			root:     bytesutil.ToBytes32([]byte{'A'}),
			name:     "happy path",
		},
		{
			sbc: &stateBalanceCache{
				stateGen: &mockStateByRooter{
					state: testStateFixture(
						testStateWithSlot(99),
						testStateWithValidators(allValidValidators)),
				},
			},
			balances: allValidBalances,
			root:     [32]byte{},
			name:     "zero root",
		},
	}
	ctx := context.Background()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cache := c.sbc
			cacheRootStart := cache.root
			b, err := cache.get(ctx, c.root)
			require.ErrorIs(t, err, c.err)
			require.DeepEqual(t, c.balances, b)
			if c.err != nil {
				// if there was an error somewhere, the root should not have changed (unless it already matched)
				require.Equal(t, cacheRootStart, cache.root)
			} else {
				// when successful, the cache should always end with a root matching the request
				require.Equal(t, c.root, cache.root)
			}
		})
	}
}
