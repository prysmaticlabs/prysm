package protoarray

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestForkChoice_BoostProposerRoot(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.SecondsPerSlot = 6
	cfg.IntervalsPerSlot = 3
	params.OverrideBeaconConfig(cfg)
	ctx := context.Background()

	t.Run("does not boost block from different slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		blockRoot := [32]byte{'A'}

		// Trying to boost a block from slot 0 should not work.
		err := f.BoostProposerRoot(ctx, 0 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{}, f.store.proposerBoostRoot)
	})
	t.Run("does not boost untimely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + X where X > attesting interval.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		attestingInterval := time.Duration(cfg.SecondsPerSlot / cfg.IntervalsPerSlot)
		greaterThanAttestingInterval := attestingInterval + 100*time.Millisecond
		genesis = genesis.Add(-greaterThanAttestingInterval * time.Second)
		blockRoot := [32]byte{'A'}

		// Trying to boost a block from slot 1 that is untimely should not work.
		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{}, f.store.proposerBoostRoot)
	})
	t.Run("boosts perfectly timely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + 0 seconds into the attesting interval.
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		fmt.Println(genesis)
		blockRoot := [32]byte{'A'}

		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{'A'}, f.store.proposerBoostRoot)
	})
	t.Run("boosts timely block from same slot", func(t *testing.T) {
		f := &ForkChoice{
			store: &Store{},
		}
		// Genesis set to 1 slot ago + (attesting interval / 2).
		genesis := time.Now().Add(-time.Duration(cfg.SecondsPerSlot) * time.Second)
		blockRoot := [32]byte{'A'}
		halfAttestingInterval := time.Second
		genesis = genesis.Add(-halfAttestingInterval)

		err := f.BoostProposerRoot(ctx, 1 /* slot */, blockRoot, genesis)
		require.NoError(t, err)
		require.DeepEqual(t, [32]byte{'A'}, f.store.proposerBoostRoot)
	})
}

func TestForkChoice_computeProposerBoostScore(t *testing.T) {
	t.Run("nil justified balances throws error", func(t *testing.T) {
		_, err := computeProposerBoostScore(nil)
		require.ErrorContains(t, "no active validators", err)
	})
	t.Run("normal active balances computes score", func(t *testing.T) {
		validatorBalances := make([]uint64, 32)
		for i := 0; i < len(validatorBalances); i++ {
			validatorBalances[i] = 10
		}
		// Avg balance is 10, and the number of validators is 32.
		// With a committee size of num validators (32) / slots per epoch (32) == 1,
		// we then have a committee weight of avg balance * committee size = 10 * 1 = 10.
		// The score then becomes 10 * PROPOSER_SCORE_BOOST // 100, which is
		// 10 * 70 / 100 = 7.
		score, err := computeProposerBoostScore(validatorBalances)
		require.NoError(t, err)
		require.Equal(t, uint64(7), score)
	})
}
