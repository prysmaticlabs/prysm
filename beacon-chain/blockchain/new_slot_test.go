package blockchain

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
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
	fcs := protoarray.New(0, 0)
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

	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 0, [32]byte{}, [32]byte{}, [32]byte{}, 0, 0))        // genesis
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 32, [32]byte{'a'}, [32]byte{}, [32]byte{}, 0, 0))    // finalized
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 64, [32]byte{'b'}, [32]byte{'a'}, [32]byte{}, 0, 0)) // justified
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 96, bj, [32]byte{'a'}, [32]byte{}, 0, 0))            // best justified
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 97, [32]byte{'d'}, [32]byte{}, [32]byte{}, 0, 0))    // bad

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
			require.DeepSSZEqual(t, service.store.BestJustifiedCheckpt(), service.store.JustifiedCheckpt())
		} else {
			require.DeepNotSSZEqual(t, service.store.BestJustifiedCheckpt(), service.store.JustifiedCheckpt())
		}
	}
}

func TestNewSlot_DontUpdateForkchoiceJustification(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	fcs := doublylinkedtree.New(0, 0)
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
	jr, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 0, [32]byte{}, [32]byte{}, [32]byte{}, 0, 0))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 32, [32]byte{'a'}, [32]byte{}, [32]byte{}, 0, 0))
	require.NoError(t, fcs.InsertOptimisticBlock(ctx, 64, jr, [32]byte{'a'}, [32]byte{}, 0, 0))
	gcp := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:], Epoch: 0}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	s := store.New(gcp, gcp)

	bcp := &ethpb.Checkpoint{Root: jr[:], Epoch: 2}
	s.SetBestJustifiedCheckpt(bcp)
	service.store = s

	headRoot, err := fcs.Head(ctx, jr, []uint64{})
	require.NoError(t, err)
	require.Equal(t, jr, headRoot)

	require.NoError(t, service.NewSlot(ctx, types.Slot(64)))
	headRoot, err = fcs.Head(ctx, jr, []uint64{})
	require.NoError(t, err)
	require.Equal(t, jr, headRoot)
}
