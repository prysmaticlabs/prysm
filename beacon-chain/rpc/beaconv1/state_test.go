package beaconv1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ptypes "github.com/gogo/protobuf/types"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	chainMock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetGenesis(t *testing.T) {
	ctx := context.Background()
	params.SetupTestConfigCleanup(t)
	config := params.BeaconConfig()
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
		resp, err := s.GetGenesis(ctx, &ptypes.Empty{})
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
		_, err := s.GetGenesis(ctx, &ptypes.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})

	t.Run("No genesis validator root", func(t *testing.T) {
		chainService := &chainMock.ChainService{
			Genesis:        genesis,
			ValidatorsRoot: [32]byte{},
		}
		s := Server{
			GenesisTimeFetcher: chainService,
			ChainInfoFetcher:   chainService,
		}
		_, err := s.GetGenesis(ctx, &ptypes.Empty{})
		assert.ErrorContains(t, "Chain genesis info is not yet known", err)
	})
}

func TestGetStateRoot(t *testing.T) {
	db := testDB.SetupDB(t)
	ctx := context.Background()

	t.Run("Head", func(t *testing.T) {
		b := testutil.NewBeaconBlock()
		b.Block.StateRoot = bytesutil.PadTo([]byte("head"), 32)
		s := Server{
			ChainInfoFetcher: &chainMock.ChainService{Block: b},
		}

		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("head"), 32), resp.Data.StateRoot)
	})

	t.Run("Genesis", func(t *testing.T) {
		b := testutil.NewBeaconBlock()
		b.Block.StateRoot = bytesutil.PadTo([]byte("genesis"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
		s := Server{
			BeaconDB: db,
		}

		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("genesis"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("genesis"), 32), resp.Data.StateRoot)
	})

	t.Run("Finalized", func(t *testing.T) {
		parent := testutil.NewBeaconBlock()
		parentR, err := parent.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, parent))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, parentR))
		b := testutil.NewBeaconBlock()
		b.Block.ParentRoot = parentR[:]
		b.Block.StateRoot = bytesutil.PadTo([]byte("finalized"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveFinalizedCheckpoint(ctx, &eth.Checkpoint{Root: r[:]}))
		s := Server{
			BeaconDB: db,
		}

		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("finalized"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.StateRoot)
	})

	t.Run("Justified", func(t *testing.T) {
		parent := testutil.NewBeaconBlock()
		parentR, err := parent.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, parent))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, parentR))
		b := testutil.NewBeaconBlock()
		b.Block.ParentRoot = parentR[:]
		b.Block.StateRoot = bytesutil.PadTo([]byte("justified"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveJustifiedCheckpoint(ctx, &eth.Checkpoint{Root: r[:]}))
		s := Server{
			BeaconDB: db,
		}

		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("justified"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("justified"), 32), resp.Data.StateRoot)
	})

	t.Run("Hex root", func(t *testing.T) {
		state, err := testutil.NewBeaconState(testutil.FillRootsNaturalOpt)
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.NoError(t, err)
		assert.DeepEqual(t, stateId, resp.Data.StateRoot)
	})

	t.Run("Hex root not found", func(t *testing.T) {
		state, err := testutil.NewBeaconState()
		require.NoError(t, err)
		chainService := &chainMock.ChainService{
			State: state,
		}
		s := Server{
			ChainInfoFetcher: chainService,
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		assert.ErrorContains(t, fmt.Sprintf("state not found in the last %d state roots in head state", len(state.StateRoots())), err)
	})

	t.Run("Slot", func(t *testing.T) {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = 100
		b.Block.StateRoot = bytesutil.PadTo([]byte("slot"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		s := Server{
			BeaconDB:           db,
			GenesisTimeFetcher: &chainMock.ChainService{},
		}

		resp, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("100"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("slot"), 32), resp.Data.StateRoot)
	})

	t.Run("Multiple slots", func(t *testing.T) {
		b := testutil.NewBeaconBlock()
		b.Block.Slot = 100
		b.Block.StateRoot = bytesutil.PadTo([]byte("slot"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		b = testutil.NewBeaconBlock()
		b.Block.Slot = 100
		b.Block.StateRoot = bytesutil.PadTo([]byte("sLot"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		s := Server{
			BeaconDB:           db,
			GenesisTimeFetcher: &chainMock.ChainService{},
		}

		_, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("100"),
		})
		assert.ErrorContains(t, "multiple blocks exist in same slot", err)
	})

	t.Run("Slot too big", func(t *testing.T) {
		s := Server{
			GenesisTimeFetcher: &chainMock.ChainService{
				Genesis: time.Now(),
			},
		}
		_, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(1, 10)),
		})
		assert.ErrorContains(t, "slot cannot be in the future", err)
	})

	t.Run("Invalid state", func(t *testing.T) {
		s := Server{}
		_, err := s.GetStateRoot(ctx, &ethpb.StateRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}

func TestGetStateFork(t *testing.T) {
	ctx := context.Background()

	fillFork := func(state *pb.BeaconState) error {
		state.Fork = &pb.Fork{
			PreviousVersion: []byte("prev"),
			CurrentVersion:  []byte("curr"),
			Epoch:           123,
		}
		return nil
	}
	headSlot := types.Slot(123)
	fillSlot := func(state *pb.BeaconState) error {
		state.Slot = headSlot
		return nil
	}
	state, err := testutil.NewBeaconState(testutil.FillRootsNaturalOpt, fillFork, fillSlot)
	require.NoError(t, err)
	stateRoot, err := state.HashTreeRoot(ctx)
	require.NoError(t, err)

	t.Run("Head", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Genesis", func(t *testing.T) {
		db := testDB.SetupDB(t)
		b := testutil.NewBeaconBlock()
		b.Block.StateRoot = bytesutil.PadTo([]byte("genesis"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
		st, err := testutil.NewBeaconState(func(state *pb.BeaconState) error {
			state.Fork = &pb.Fork{
				PreviousVersion: []byte("prev"),
				CurrentVersion:  []byte("curr"),
				Epoch:           123,
			}
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, db.SaveState(ctx, st, r))

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				BeaconDB: db,
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte("genesis"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Finalized", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{
					FinalizedCheckPoint: &eth.Checkpoint{
						Root: stateRoot[:],
					},
				},
				StateGenService: stateGen,
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte("finalized"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Justified", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{
					CurrentJustifiedCheckPoint: &eth.Checkpoint{
						Root: stateRoot[:],
					},
				},
				StateGenService: stateGen,
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte("justified"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Hex root", func(t *testing.T) {
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[bytesutil.ToBytes32(stateId)] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
				StateGenService:  stateGen,
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Slot", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesBySlot[headSlot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				GenesisTimeFetcher: &chainMock.ChainService{Slot: &headSlot},
				StateGenService:    stateGen,
			},
		}

		resp, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(uint64(headSlot), 10)),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, []byte("prev"), resp.Data.PreviousVersion)
		assert.DeepEqual(t, []byte("curr"), resp.Data.CurrentVersion)
		assert.Equal(t, types.Epoch(123), resp.Data.Epoch)
	})

	t.Run("Slot too big", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				GenesisTimeFetcher: &chainMock.ChainService{
					Genesis: time.Now(),
				},
			},
		}
		_, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(1, 10)),
		})
		assert.ErrorContains(t, "slot cannot be in the future", err)
	})

	t.Run("Invalid state", func(t *testing.T) {
		s := Server{}
		_, err := s.GetStateFork(ctx, &ethpb.StateRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}

func TestGetFinalityCheckpoints(t *testing.T) {
	ctx := context.Background()

	fillCheckpoints := func(state *pb.BeaconState) error {
		state.PreviousJustifiedCheckpoint = &eth.Checkpoint{
			Root:  bytesutil.PadTo([]byte("previous"), 32),
			Epoch: 113,
		}
		state.CurrentJustifiedCheckpoint = &eth.Checkpoint{
			Root:  bytesutil.PadTo([]byte("current"), 32),
			Epoch: 123,
		}
		state.FinalizedCheckpoint = &eth.Checkpoint{
			Root:  bytesutil.PadTo([]byte("finalized"), 32),
			Epoch: 103,
		}
		return nil
	}
	headSlot := types.Slot(123)
	fillSlot := func(state *pb.BeaconState) error {
		state.Slot = headSlot
		return nil
	}
	state, err := testutil.NewBeaconState(testutil.FillRootsNaturalOpt, fillCheckpoints, fillSlot)
	require.NoError(t, err)
	stateRoot, err := state.HashTreeRoot(ctx)
	require.NoError(t, err)

	t.Run("Head", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Genesis", func(t *testing.T) {
		db := testDB.SetupDB(t)
		b := testutil.NewBeaconBlock()
		b.Block.StateRoot = bytesutil.PadTo([]byte("genesis"), 32)
		require.NoError(t, db.SaveBlock(ctx, b))
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveStateSummary(ctx, &pb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
		st, err := testutil.NewBeaconState(func(state *pb.BeaconState) error {
			state.PreviousJustifiedCheckpoint = &eth.Checkpoint{
				Root:  bytesutil.PadTo([]byte("previous"), 32),
				Epoch: 113,
			}
			state.CurrentJustifiedCheckpoint = &eth.Checkpoint{
				Root:  bytesutil.PadTo([]byte("current"), 32),
				Epoch: 123,
			}
			state.FinalizedCheckpoint = &eth.Checkpoint{
				Root:  bytesutil.PadTo([]byte("finalized"), 32),
				Epoch: 103,
			}
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, db.SaveState(ctx, st, r))

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				BeaconDB: db,
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("genesis"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Finalized", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{
					FinalizedCheckPoint: &eth.Checkpoint{
						Root: stateRoot[:],
					},
				},
				StateGenService: stateGen,
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("finalized"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Justified", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{
					CurrentJustifiedCheckPoint: &eth.Checkpoint{
						Root: stateRoot[:],
					},
				},
				StateGenService: stateGen,
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("justified"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Hex root", func(t *testing.T) {
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		stateGen := stategen.NewMockService()
		stateGen.StatesByRoot[bytesutil.ToBytes32(stateId)] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
				StateGenService:  stateGen,
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Hex root not found", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: state},
			},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: stateId,
		})
		require.ErrorContains(t, "state not found in the last 8192 state roots in head state", err)
	})

	t.Run("Slot", func(t *testing.T) {
		stateGen := stategen.NewMockService()
		stateGen.StatesBySlot[headSlot] = state

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				GenesisTimeFetcher: &chainMock.ChainService{Slot: &headSlot},
				StateGenService:    stateGen,
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(uint64(headSlot), 10)),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("previous"), 32), resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(113), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("current"), 32), resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(123), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, bytesutil.PadTo([]byte("finalized"), 32), resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(103), resp.Data.Finalized.Epoch)
	})

	t.Run("Slot too big", func(t *testing.T) {
		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				GenesisTimeFetcher: &chainMock.ChainService{
					Genesis: time.Now(),
				},
			},
		}
		_, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte(strconv.FormatUint(1, 10)),
		})
		assert.ErrorContains(t, "slot cannot be in the future", err)
	})

	t.Run("Checkpoints not available", func(t *testing.T) {
		st, err := testutil.NewBeaconState()
		require.NoError(t, err)
		err = st.SetPreviousJustifiedCheckpoint(nil)
		require.NoError(t, err)
		err = st.SetCurrentJustifiedCheckpoint(nil)
		require.NoError(t, err)
		err = st.SetFinalizedCheckpoint(nil)
		require.NoError(t, err)

		s := Server{
			StateFetcher: statefetcher.StateFetcher{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
		}

		resp, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], resp.Data.PreviousJustified.Root)
		assert.Equal(t, types.Epoch(0), resp.Data.PreviousJustified.Epoch)
		assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], resp.Data.CurrentJustified.Root)
		assert.Equal(t, types.Epoch(0), resp.Data.CurrentJustified.Epoch)
		assert.DeepEqual(t, params.BeaconConfig().ZeroHash[:], resp.Data.Finalized.Root)
		assert.Equal(t, types.Epoch(0), resp.Data.Finalized.Epoch)
	})

	t.Run("Invalid state", func(t *testing.T) {
		s := Server{}
		_, err := s.GetFinalityCheckpoints(ctx, &ethpb.StateRequest{
			StateId: []byte("foo"),
		})
		require.ErrorContains(t, "invalid state ID: foo", err)
	})
}
