package debug

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	blockchainmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetBeaconStateV2(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	t.Run("Phase 0", func(t *testing.T) {
		fakeState, err := util.NewBeaconState()
		require.NoError(t, err)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           &blockchainmock.ChainService{},
			OptimisticModeFetcher: &blockchainmock.ChainService{},
			BeaconDB:              db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateAltair(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           &blockchainmock.ChainService{},
			OptimisticModeFetcher: &blockchainmock.ChainService{},
			BeaconDB:              db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           &blockchainmock.ChainService{},
			OptimisticModeFetcher: &blockchainmock.ChainService{},
			BeaconDB:              db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           &blockchainmock.ChainService{},
			OptimisticModeFetcher: &blockchainmock.ChainService{Optimistic: true},
			BeaconDB:              db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetBeaconStateSSZ(t *testing.T) {
	fakeState, err := util.NewBeaconState()
	require.NoError(t, err)
	sszState, err := fakeState.MarshalSSZ()
	require.NoError(t, err)

	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: fakeState,
		},
	}
	resp, err := server.GetBeaconStateSSZ(context.Background(), &ethpbv1.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)

	assert.DeepEqual(t, sszState, resp.Data)
}

func TestGetBeaconStateSSZV2(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		fakeState, err := util.NewBeaconState()
		require.NoError(t, err)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateAltair(t, 1)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.BeaconStateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
}

func TestListForkChoiceHeadsV2(t *testing.T) {
	ctx := context.Background()

	expectedSlotsAndRoots := []struct {
		Slot types.Slot
		Root [32]byte
	}{{
		Slot: 0,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("foo"), 32)),
	}, {
		Slot: 1,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("bar"), 32)),
	}}

	chainService := &blockchainmock.ChainService{}
	server := &Server{
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	resp, err := server.ListForkChoiceHeadsV2(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp.Data))
	for _, sr := range expectedSlotsAndRoots {
		found := false
		for _, h := range resp.Data {
			if h.Slot == sr.Slot {
				found = true
				assert.DeepEqual(t, sr.Root[:], h.Root)
			}
			assert.Equal(t, false, h.ExecutionOptimistic)
		}
		assert.Equal(t, true, found, "Expected head not found")
	}

	t.Run("optimistic head", func(t *testing.T) {
		chainService := &blockchainmock.ChainService{Optimistic: true}
		server := &Server{
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		resp, err := server.ListForkChoiceHeadsV2(ctx, &emptypb.Empty{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
		for _, sr := range expectedSlotsAndRoots {
			found := false
			for _, h := range resp.Data {
				if h.Slot == sr.Slot {
					found = true
					assert.DeepEqual(t, sr.Root[:], h.Root)
				}
				assert.Equal(t, true, h.ExecutionOptimistic)
			}
			assert.Equal(t, true, found, "Expected head not found")
		}
	})
}

func TestServer_GetForkChoice(t *testing.T) {
	store := doublylinkedtree.New()
	fRoot := [32]byte{'a'}
	jRoot := [32]byte{'b'}
	fc := &forkchoicetypes.Checkpoint{Epoch: 2, Root: fRoot}
	jc := &forkchoicetypes.Checkpoint{Epoch: 3, Root: jRoot}
	require.NoError(t, store.UpdateFinalizedCheckpoint(fc))
	require.NoError(t, store.UpdateJustifiedCheckpoint(jc))
	bs := &Server{ForkFetcher: &blockchainmock.ChainService{ForkChoiceStore: store}}
	res, err := bs.GetForkChoice(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	require.Equal(t, types.Epoch(3), res.JustifiedCheckpoint.Epoch, "Did not get wanted justified epoch")
	require.Equal(t, types.Epoch(2), res.FinalizedCheckpoint.Epoch, "Did not get wanted finalized epoch")
}
