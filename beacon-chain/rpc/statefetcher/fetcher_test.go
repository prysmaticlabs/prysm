package statefetcher

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	statenative "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"

	"github.com/ethereum/go-ethereum/common/hexutil"
	chainMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	mockstategen "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen/mock"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
)

func TestGetState(t *testing.T) {
	ctx := context.Background()

	headSlot := types.Slot(123)
	fillSlot := func(state *ethpb.BeaconState) error {
		state.Slot = headSlot
		return nil
	}
	newBeaconState, err := util.NewBeaconState(util.FillRootsNaturalOpt, fillSlot)
	require.NoError(t, err)
	stateRoot, err := newBeaconState.HashTreeRoot(ctx)
	require.NoError(t, err)

	t.Run("head", func(t *testing.T) {
		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{State: newBeaconState},
		}

		s, err := p.State(ctx, []byte("head"))
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.DeepEqual(t, stateRoot, sRoot)
	})

	t.Run("genesis", func(t *testing.T) {
		params.SetupTestConfigCleanup(t)
		cfg := params.BeaconConfig().Copy()
		cfg.ConfigName = "test"
		params.OverrideBeaconConfig(cfg)

		db := testDB.SetupDB(t)
		b := util.NewBeaconBlock()
		b.Block.StateRoot = bytesutil.PadTo([]byte("foo"), 32)
		util.SaveBlock(t, ctx, db, b)
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)

		bs, err := util.NewBeaconState(func(state *ethpb.BeaconState) error {
			state.BlockRoots[0] = r[:]
			return nil
		})
		require.NoError(t, err)
		newStateRoot, err := bs.HashTreeRoot(ctx)
		require.NoError(t, err)

		require.NoError(t, db.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
		require.NoError(t, db.SaveState(ctx, bs, r))

		cc := &mockstategen.MockCanonicalChecker{Is: true}
		cs := &mockstategen.MockCurrentSlotter{Slot: bs.Slot() + 1}
		ch := stategen.NewCanonicalHistory(db, cc, cs)
		currentSlot := types.Slot(0)
		p := StateProvider{
			BeaconDB:           db,
			ReplayerBuilder:    ch,
			GenesisTimeFetcher: &chainMock.ChainService{Slot: &currentSlot},
			ChainInfoFetcher:   &chainMock.ChainService{State: bs},
		}

		s, err := p.State(ctx, []byte("genesis"))
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.DeepEqual(t, newStateRoot, sRoot)
	})

	t.Run("finalized", func(t *testing.T) {
		stateGen := mockstategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = newBeaconState

		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{
				FinalizedCheckPoint: &ethpb.Checkpoint{
					Root: stateRoot[:],
				},
			},
			StateGenService: stateGen,
		}

		s, err := p.State(ctx, []byte("finalized"))
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.Equal(t, stateRoot, sRoot)
	})

	t.Run("justified", func(t *testing.T) {
		stateGen := mockstategen.NewMockService()
		stateGen.StatesByRoot[stateRoot] = newBeaconState

		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{
				CurrentJustifiedCheckPoint: &ethpb.Checkpoint{
					Root: stateRoot[:],
				},
			},
			StateGenService: stateGen,
		}

		s, err := p.State(ctx, []byte("justified"))
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.DeepEqual(t, stateRoot, sRoot)
	})

	t.Run("hex_root", func(t *testing.T) {
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)
		stateGen := mockstategen.NewMockService()
		stateGen.StatesByRoot[bytesutil.ToBytes32(stateId)] = newBeaconState

		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{State: newBeaconState},
			StateGenService:  stateGen,
		}

		s, err := p.State(ctx, stateId)
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.DeepEqual(t, stateRoot, sRoot)
	})

	t.Run("hex_root_not_found", func(t *testing.T) {
		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{State: newBeaconState},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = p.State(ctx, stateId)
		require.ErrorContains(t, "state not found in the last 8192 state roots", err)
	})

	t.Run("slot", func(t *testing.T) {
		p := StateProvider{
			GenesisTimeFetcher: &chainMock.ChainService{Slot: &headSlot},
			ChainInfoFetcher: &chainMock.ChainService{
				CanonicalRoots: map[[32]byte]bool{
					bytesutil.ToBytes32(newBeaconState.LatestBlockHeader().ParentRoot): true,
				},
				State: newBeaconState,
			},
			ReplayerBuilder: mockstategen.NewMockReplayerBuilder(mockstategen.WithMockState(newBeaconState)),
		}

		s, err := p.State(ctx, []byte(strconv.FormatUint(uint64(headSlot), 10)))
		require.NoError(t, err)
		sRoot, err := s.HashTreeRoot(ctx)
		require.NoError(t, err)
		assert.Equal(t, stateRoot, sRoot)
	})

	t.Run("invalid_state", func(t *testing.T) {
		p := StateProvider{}
		_, err := p.State(ctx, []byte("foo"))
		require.ErrorContains(t, "could not parse state ID", err)
	})
}

func TestGetStateRoot(t *testing.T) {
	ctx := context.Background()

	headSlot := types.Slot(123)
	fillSlot := func(state *ethpb.BeaconState) error {
		state.Slot = headSlot
		return nil
	}
	newBeaconState, err := util.NewBeaconState(util.FillRootsNaturalOpt, fillSlot)
	require.NoError(t, err)
	stateRoot, err := newBeaconState.HashTreeRoot(ctx)
	require.NoError(t, err)

	t.Run("head", func(t *testing.T) {
		b := util.NewBeaconBlock()
		b.Block.StateRoot = stateRoot[:]
		wsb, err := blocks.NewSignedBeaconBlock(b)
		require.NoError(t, err)
		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{
				State: newBeaconState,
				Block: wsb,
			},
		}

		s, err := p.StateRoot(ctx, []byte("head"))
		require.NoError(t, err)
		assert.DeepEqual(t, stateRoot[:], s)
	})

	t.Run("genesis", func(t *testing.T) {
		db := testDB.SetupDB(t)
		b := util.NewBeaconBlock()
		util.SaveBlock(t, ctx, db, b)
		r, err := b.Block.HashTreeRoot()
		require.NoError(t, err)

		bs, err := util.NewBeaconState(func(state *ethpb.BeaconState) error {
			state.BlockRoots[0] = r[:]
			return nil
		})
		require.NoError(t, err)

		require.NoError(t, db.SaveStateSummary(ctx, &ethpb.StateSummary{Root: r[:]}))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, r))
		require.NoError(t, db.SaveState(ctx, bs, r))

		p := StateProvider{
			BeaconDB: db,
		}

		s, err := p.StateRoot(ctx, []byte("genesis"))
		require.NoError(t, err)
		genesisBlock, err := db.GenesisBlock(ctx)
		require.NoError(t, err)
		assert.DeepEqual(t, genesisBlock.Block().StateRoot(), s)
	})

	t.Run("finalized", func(t *testing.T) {
		db := testDB.SetupDB(t)
		genesis := bytesutil.ToBytes32([]byte("genesis"))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = genesis[:]
		blk.Block.Slot = 40
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		cp := &ethpb.Checkpoint{
			Epoch: 5,
			Root:  root[:],
		}
		// a valid chain is required to save finalized checkpoint.
		util.SaveBlock(t, ctx, db, blk)
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(1))
		// a state is required to save checkpoint
		require.NoError(t, db.SaveState(ctx, st, root))
		require.NoError(t, db.SaveFinalizedCheckpoint(ctx, cp))

		p := StateProvider{
			BeaconDB: db,
		}

		s, err := p.StateRoot(ctx, []byte("finalized"))
		require.NoError(t, err)
		assert.DeepEqual(t, blk.Block.StateRoot, s)
	})

	t.Run("justified", func(t *testing.T) {
		db := testDB.SetupDB(t)
		genesis := bytesutil.ToBytes32([]byte("genesis"))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = genesis[:]
		blk.Block.Slot = 40
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		cp := &ethpb.Checkpoint{
			Epoch: 5,
			Root:  root[:],
		}
		// a valid chain is required to save finalized checkpoint.
		util.SaveBlock(t, ctx, db, blk)
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(1))
		// a state is required to save checkpoint
		require.NoError(t, db.SaveState(ctx, st, root))
		require.NoError(t, db.SaveJustifiedCheckpoint(ctx, cp))

		p := StateProvider{
			BeaconDB: db,
		}

		s, err := p.StateRoot(ctx, []byte("justified"))
		require.NoError(t, err)
		assert.DeepEqual(t, blk.Block.StateRoot, s)
	})

	t.Run("hex_root", func(t *testing.T) {
		stateId, err := hexutil.Decode("0x" + strings.Repeat("0", 63) + "1")
		require.NoError(t, err)

		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{State: newBeaconState},
		}

		s, err := p.StateRoot(ctx, stateId)
		require.NoError(t, err)
		assert.DeepEqual(t, stateId, s)
	})

	t.Run("hex_root_not_found", func(t *testing.T) {
		p := StateProvider{
			ChainInfoFetcher: &chainMock.ChainService{State: newBeaconState},
		}
		stateId, err := hexutil.Decode("0x" + strings.Repeat("f", 64))
		require.NoError(t, err)
		_, err = p.StateRoot(ctx, stateId)
		require.ErrorContains(t, "state root not found in the last 8192 state roots", err)
	})

	t.Run("slot", func(t *testing.T) {
		db := testDB.SetupDB(t)
		genesis := bytesutil.ToBytes32([]byte("genesis"))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesis))
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = genesis[:]
		blk.Block.Slot = 40
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(1))
		// a state is required to save checkpoint
		require.NoError(t, db.SaveState(ctx, st, root))

		slot := types.Slot(40)
		p := StateProvider{
			GenesisTimeFetcher: &chainMock.ChainService{Slot: &slot},
			BeaconDB:           db,
		}

		s, err := p.StateRoot(ctx, []byte(strconv.FormatUint(uint64(slot), 10)))
		require.NoError(t, err)
		assert.DeepEqual(t, blk.Block.StateRoot, s)
	})

	t.Run("slot_too_big", func(t *testing.T) {
		p := StateProvider{
			GenesisTimeFetcher: &chainMock.ChainService{
				Genesis: time.Now(),
			},
		}
		_, err := p.StateRoot(ctx, []byte(strconv.FormatUint(1, 10)))
		assert.ErrorContains(t, "slot cannot be in the future", err)
	})

	t.Run("invalid_state", func(t *testing.T) {
		p := StateProvider{}
		_, err := p.StateRoot(ctx, []byte("foo"))
		require.ErrorContains(t, "could not parse state ID", err)
	})
}

func TestNewStateNotFoundError(t *testing.T) {
	e := NewStateNotFoundError(100)
	assert.Equal(t, "state not found in the last 100 state roots", e.message)
}

func TestStateBySlot_FutureSlot(t *testing.T) {
	slot := types.Slot(100)
	p := StateProvider{GenesisTimeFetcher: &chainMock.ChainService{Slot: &slot}}
	_, err := p.StateBySlot(context.Background(), 101)
	assert.ErrorContains(t, "requested slot is in the future", err)
}

func TestStateBySlot_AfterHeadSlot(t *testing.T) {
	st, err := statenative.InitializeFromProtoPhase0(&ethpb.BeaconState{Slot: 100})
	require.NoError(t, err)
	currentSlot := types.Slot(102)
	mock := &chainMock.ChainService{State: st, Slot: &currentSlot}
	p := StateProvider{ChainInfoFetcher: mock, GenesisTimeFetcher: mock}
	_, err = p.StateBySlot(context.Background(), 101)
	assert.ErrorContains(t, "requested slot number is higher than head slot number", err)
}
