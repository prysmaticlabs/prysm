package blockchain

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestService_newSlot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New(0, 0)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	ctx := context.Background()

	require.NoError(t, fcs.ProcessBlock(ctx, 0, [32]byte{}, [32]byte{}, 0, 0, true))        // genesis
	require.NoError(t, fcs.ProcessBlock(ctx, 32, [32]byte{'a'}, [32]byte{}, 0, 0, true))    // finalized
	require.NoError(t, fcs.ProcessBlock(ctx, 64, [32]byte{'b'}, [32]byte{'a'}, 0, 0, true)) // justified
	require.NoError(t, fcs.ProcessBlock(ctx, 96, [32]byte{'c'}, [32]byte{'a'}, 0, 0, true)) // best justified
	require.NoError(t, fcs.ProcessBlock(ctx, 97, [32]byte{'d'}, [32]byte{}, 0, 0, true))    // bad

	type args struct {
		slot          types.Slot
		finalized     *ethpb.Checkpoint
		justified     *ethpb.Checkpoint
		bestJustified *ethpb.Checkpoint
		shouldEqual   bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Not epoch boundary. No change",
			args: args{
				slot:          params.BeaconConfig().SlotsPerEpoch + 1,
				finalized:     &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'a'}, 32)},
				justified:     &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'b'}, 32)},
				bestJustified: &ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'c'}, 32)},
				shouldEqual:   false,
			},
		},
		{
			name: "Justified higher than best justified. No change",
			args: args{
				slot:          params.BeaconConfig().SlotsPerEpoch,
				finalized:     &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'a'}, 32)},
				justified:     &ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'b'}, 32)},
				bestJustified: &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'c'}, 32)},
				shouldEqual:   false,
			},
		},
		{
			name: "Best justified not on the same chain as finalized. No change",
			args: args{
				slot:          params.BeaconConfig().SlotsPerEpoch,
				finalized:     &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'a'}, 32)},
				justified:     &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'b'}, 32)},
				bestJustified: &ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'d'}, 32)},
				shouldEqual:   false,
			},
		},
		{
			name: "Best justified on the same chain as finalized. Yes change",
			args: args{
				slot:          params.BeaconConfig().SlotsPerEpoch,
				finalized:     &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'a'}, 32)},
				justified:     &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'b'}, 32)},
				bestJustified: &ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'c'}, 32)},
				shouldEqual:   true,
			},
		},
	}
	for _, test := range tests {
		service, err := NewService(ctx, opts...)
		require.NoError(t, err)
		store := store.New(test.args.justified, test.args.finalized)
		store.SetBestJustifiedCheckpt(test.args.bestJustified)
		service.store = store

		require.NoError(t, service.NewSlot(ctx, test.args.slot))
		if test.args.shouldEqual {
			require.DeepSSZEqual(t, service.store.BestJustifiedCheckpt(), service.store.JustifiedCheckpt())
		} else {
			require.DeepNotSSZEqual(t, service.store.BestJustifiedCheckpt(), service.store.JustifiedCheckpt())
		}
	}
}
