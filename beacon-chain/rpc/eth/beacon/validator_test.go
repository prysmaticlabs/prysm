package beacon

import (
	"context"
	"testing"

	chainMock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

func TestListCommittees(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 8192)
	epoch := slots.ToEpoch(st.Slot())

	t.Run("Head All Committees", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch)*2, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Index == primitives.CommitteeIndex(0) || datum.Index == primitives.CommitteeIndex(1))
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
		}
	})

	t.Run("Head All Committees of Epoch 10", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}
		epoch := primitives.Epoch(10)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Epoch:   &epoch,
		})
		require.NoError(t, err)
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Slot >= 320 && datum.Slot <= 351)
		}
	})

	t.Run("Head All Committees of Slot 4", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		slot := primitives.Slot(4)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
		index := primitives.CommitteeIndex(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			index++
		}
	})

	t.Run("Head All Committees of Index 1", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		index := primitives.CommitteeIndex(1)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch), len(resp.Data))
		slot := primitives.Slot(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			slot++
		}
	})

	t.Run("Head All Committees of Slot 2, Index 1", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		index := primitives.CommitteeIndex(1)
		slot := primitives.Slot(2)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
		}
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		headerRoot, err := st.LatestBlockHeader().HashTreeRoot()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			FinalizedRoots: map[[32]byte]bool{
				headerRoot: true,
			},
		}
		s := Server{
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.Finalized)
	})
}
