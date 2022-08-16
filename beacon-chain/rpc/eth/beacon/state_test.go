package beacon

import (
	"context"
	"testing"
	"time"

	chainMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetGenesis(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig().Copy()
	config.GenesisForkVersion = []byte("genesis")
	params.OverrideBeaconConfig(config)
	genesis := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	validatorsRoot := [32]byte{1, 2, 3, 4, 5, 6}

	t.Run("OK", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        genesis,
			ValidatorsRoot: validatorsRoot,
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		resp, err := s.GetGenesis(ctx, &emptypb.Empty{})
		require.NoError(t, err)
		assert.Equal(t, genesis.Unix(), resp.Data.GenesisTime.Seconds)
		assert.Equal(t, int32(0), resp.Data.GenesisTime.Nanos)
		assert.DeepEqual(t, validatorsRoot[:], resp.Data.GenesisValidatorsRoot)
		assert.DeepEqual(t, []byte("genesis"), resp.Data.GenesisForkVersion)
	})

	t.Run("No genesis time", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        time.Time{},
			ValidatorsRoot: validatorsRoot,
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		_, err := s.GetGenesis(ctx, &emptypb.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})

	t.Run("No genesis validators root", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        genesis,
			ValidatorsRoot: [32]byte{},
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		_, err := s.GetGenesis(ctx, &emptypb.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})
}

func TestGetStateRoot(t *testing.T) {
	ctx := context.Background()
	fakeState, err := util.NewBeaconState()
	require.NoError(t, err)
	stateRoot, err := fakeState.HashTreeRoot(ctx)
	require.NoError(t, err)
	db := dbTest.SetupDB(t)

	chainService := &chainMock.ChainService{}
	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconStateRoot: stateRoot[:],
			BeaconState:     fakeState,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		BeaconDB:              db,
	}

	resp, err := server.GetStateRoot(context.Background(), &eth.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.DeepEqual(t, stateRoot[:], resp.Data.Root)

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconStateRoot: stateRoot[:],
				BeaconState:     fakeState,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := server.GetStateRoot(context.Background(), &eth.StateRequest{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetStateFork(t *testing.T) {
	ctx := context.Background()
	fillFork := func(state *ethpb.BeaconState) error {
		state.Fork = &ethpb.Fork{
			PreviousVersion: []byte("prev"),
			CurrentVersion:  []byte("curr"),
			Epoch:           123,
		}
		return nil
	}
	fakeState, err := util.NewBeaconState(fillFork)
	require.NoError(t, err)
	db := dbTest.SetupDB(t)

	chainService := &chainMock.ChainService{}
	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: fakeState,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		BeaconDB:              db,
	}

	resp, err := server.GetStateFork(ctx, &eth.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	expectedFork := fakeState.Fork()
	assert.Equal(t, expectedFork.Epoch, resp.Data.Epoch)
	assert.DeepEqual(t, expectedFork.CurrentVersion, resp.Data.CurrentVersion)
	assert.DeepEqual(t, expectedFork.PreviousVersion, resp.Data.PreviousVersion)

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := server.GetStateFork(context.Background(), &eth.StateRequest{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetFinalityCheckpoints(t *testing.T) {
	ctx := context.Background()
	fillCheckpoints := func(state *ethpb.BeaconState) error {
		state.PreviousJustifiedCheckpoint = &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("previous"), 32),
			Epoch: 113,
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("current"), 32),
			Epoch: 123,
		}
		state.FinalizedCheckpoint = &ethpb.Checkpoint{
			Root:  bytesutil.PadTo([]byte("finalized"), 32),
			Epoch: 103,
		}
		return nil
	}
	fakeState, err := util.NewBeaconState(fillCheckpoints)
	require.NoError(t, err)
	db := dbTest.SetupDB(t)

	chainService := &chainMock.ChainService{}
	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: fakeState,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		BeaconDB:              db,
	}

	resp, err := server.GetFinalityCheckpoints(ctx, &eth.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, fakeState.FinalizedCheckpoint().Epoch, resp.Data.Finalized.Epoch)
	assert.DeepEqual(t, fakeState.FinalizedCheckpoint().Root, resp.Data.Finalized.Root)
	assert.Equal(t, fakeState.CurrentJustifiedCheckpoint().Epoch, resp.Data.CurrentJustified.Epoch)
	assert.DeepEqual(t, fakeState.CurrentJustifiedCheckpoint().Root, resp.Data.CurrentJustified.Root)
	assert.Equal(t, fakeState.PreviousJustifiedCheckpoint().Epoch, resp.Data.PreviousJustified.Epoch)
	assert.DeepEqual(t, fakeState.PreviousJustifiedCheckpoint().Root, resp.Data.PreviousJustified.Root)

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := server.GetFinalityCheckpoints(context.Background(), &eth.StateRequest{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.DeepEqual(t, true, resp.ExecutionOptimistic)
	})
}
