package blockchain

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestService_newSlot(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	fcs := protoarray.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	ctx := context.Background()

	genesisStateRoot := [32]byte{}
	genesis := blocks.NewGenesisBlock(genesisStateRoot[:])
	wsb, err := wrapper.WrappedSignedBeaconBlock(genesis)
	require.NoError(t, err)
	assert.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	bj, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)

	state, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{}, [32]byte{}, [32]byte{}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot)) // genesis
	state, blkRoot, err = prepareForkchoiceState(ctx, 32, [32]byte{'a'}, [32]byte{}, [32]byte{}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot)) // finalized
	state, blkRoot, err = prepareForkchoiceState(ctx, 64, [32]byte{'b'}, [32]byte{'a'}, [32]byte{}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot)) // justified
	state, blkRoot, err = prepareForkchoiceState(ctx, 96, bj, [32]byte{'a'}, [32]byte{}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot)) // best justified
	state, blkRoot, err = prepareForkchoiceState(ctx, 97, [32]byte{'d'}, [32]byte{}, [32]byte{}, 0, 0)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot)) // bad

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
				bestJustified: &ethpb.Checkpoint{Epoch: 3, Root: bj[:]},
				shouldEqual:   false,
			},
		},
		{
			name: "Justified higher than best justified. No change",
			args: args{
				slot:          params.BeaconConfig().SlotsPerEpoch,
				finalized:     &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'a'}, 32)},
				justified:     &ethpb.Checkpoint{Epoch: 3, Root: bytesutil.PadTo([]byte{'b'}, 32)},
				bestJustified: &ethpb.Checkpoint{Epoch: 2, Root: bj[:]},
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
				bestJustified: &ethpb.Checkpoint{Epoch: 3, Root: bj[:]},
				shouldEqual:   true,
			},
		},
	}
	for _, test := range tests {
		service, err := NewService(ctx, opts...)
		require.NoError(t, err)
		s := store.New(test.args.justified, test.args.finalized)
		s.SetBestJustifiedCheckpt(test.args.bestJustified)
		service.store = s

		require.NoError(t, service.NewSlot(ctx, test.args.slot))
		if test.args.shouldEqual {
			bcp, err := service.store.BestJustifiedCheckpt()
			require.NoError(t, err)
			cp, err := service.store.JustifiedCheckpt()
			require.NoError(t, err)
			require.DeepSSZEqual(t, bcp, cp)
		} else {
			bcp, err := service.store.BestJustifiedCheckpt()
			require.NoError(t, err)
			cp, err := service.store.JustifiedCheckpt()
			require.NoError(t, err)
			require.DeepNotSSZEqual(t, bcp, cp)
		}
	}
}
