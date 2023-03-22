package state_native

import (
	"math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	customtypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native/custom-types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbalpha "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"golang.org/x/sync/errgroup"
)

func BenchmarkCopyBalancesFirstTime(b *testing.B) {
	raw := [51200000]uint64{}
	b.Run("512", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:512])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
	b.Run("5,120", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:512])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
	b.Run("51,200", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:5120])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
	b.Run("512,000", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:51200])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
	b.Run("5,120,000", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:512000])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
	b.Run("51,200,000", func(b *testing.B) {
		bals := customtypes.NewBalances(raw[:5120000])
		for i := 0; i < b.N; i++ {
			bals.Copy()
		}
	})
}

func BenchmarkCopyBalancesSecondTime(b *testing.B) {
	raw := [51200000]uint64{}
	b.Run("512", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:512])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
	b.Run("5,120", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:5120])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
	b.Run("51,200", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:51200])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
	b.Run("512,000", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:512000])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
	b.Run("5,120,000", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:5120000])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
	b.Run("51,200,000", func(b *testing.B) {
		bals1 := customtypes.NewBalances(raw[:51200000])
		bals2 := bals1.Copy()
		for i := 0; i < b.N; i++ {
			bals2.Copy()
		}
	})
}

func BenchmarkAppendWhenChunksAreFull(b *testing.B) {
	raw := [51200000]uint64{}
	b.Run("512", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:511])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
	b.Run("5,120", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:5119])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
	b.Run("51,200", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:51199])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
	b.Run("512,000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:511999])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
	b.Run("5,120,000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:5119999])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
	b.Run("51,200,000", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			bals := customtypes.NewBalances(raw[:51199999])
			bals.Append(0)
			b.StartTimer()
			bals.Append(0)
		}
	})
}

func TestBalancesChaos(t *testing.T) {
	raw := [100000]uint64{}
	bals := customtypes.NewBalances(raw[:])
	s, err := InitializeFromProtoUnsafePhase0(&ethpbalpha.BeaconState{})
	require.NoError(t, err)
	require.NoError(t, s.SetBalances(bals))
	states := make([]state.BeaconState, 100)
	for i := range states {
		states[i] = s.Copy()
	}

	var g errgroup.Group
	for i := 0; i < len(states); i++ {
		auxiliary := i
		g.Go(func() error {
			st := states[auxiliary]
			for j := 0; j < 100000; j++ {
				r := rand.New(rand.NewSource(int64(j)))
				choice := r.Intn(3)
				if choice == 0 {
					if err = st.AppendBalance(uint64(j)); err != nil {
						return err
					}
				} else {
					index := r.Intn(st.Balances().Len())
					if err = st.UpdateBalancesAtIndex(primitives.ValidatorIndex(index), uint64(j)); err != nil {
						return err
					}
				}
			}
			return nil
		})
	}

	assert.NoError(t, g.Wait())
}
