package blockchain

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	testDB "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v3/beacon-chain/execution/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_isNewProposer(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	require.Equal(t, false, service.isNewProposer())

	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(service.CurrentSlot()+1, 0, [8]byte{}, [32]byte{} /* root */)
	require.Equal(t, true, service.isNewProposer())
}

func TestService_isNewHead(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	require.Equal(t, true, service.isNewHead([32]byte{}))

	service.head = &head{root: [32]byte{1}}
	require.Equal(t, true, service.isNewHead([32]byte{2}))
	require.Equal(t, false, service.isNewHead([32]byte{1}))

	// Nil head should use origin root
	service.head = nil
	service.originBlockRoot = [32]byte{3}
	require.Equal(t, true, service.isNewHead([32]byte{2}))
	require.Equal(t, false, service.isNewHead([32]byte{3}))
}

func TestService_getHeadStateAndBlock(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	_, _, err := service.getStateAndBlock(context.Background(), [32]byte{})
	require.ErrorContains(t, "block does not exist", err)

	blk, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlock(&ethpb.SignedBeaconBlock{Signature: []byte{1}}))
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(context.Background(), blk))

	st, _ := util.DeterministicGenesisState(t, 1)
	r, err := blk.Block().HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(context.Background(), st, r))

	gotState, err := service.cfg.BeaconDB.State(context.Background(), r)
	require.NoError(t, err)
	require.DeepEqual(t, st.ToProto(), gotState.ToProto())

	gotBlk, err := service.cfg.BeaconDB.Block(context.Background(), r)
	require.NoError(t, err)
	require.DeepEqual(t, blk, gotBlk)
}

func TestService_forkchoiceUpdateWithExecution_exceptionalCases(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)

	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	service.cfg.ProposerSlotIndexCache = cache.NewProposerPayloadIDsCache()
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, service.headRoot()))
	hookErr := "could not notify forkchoice update"
	invalidStateErr := "could not get state summary: could not find block in DB"
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)
	gb, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, [32]byte{'a'}, gb))
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, [32]byte{'a'}))
	require.LogsContain(t, hook, invalidStateErr)

	hook.Reset()
	service.head = &head{
		root:  [32]byte{'a'},
		block: nil, /* should not panic if notify head uses correct head */
	}

	// Block in Cache
	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	wsb, err := blocks.NewSignedBeaconBlock(b)
	require.NoError(t, err)
	r1, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, service.saveInitSyncBlock(ctx, r1, wsb))
	st, _ := util.DeterministicGenesisState(t, 1)
	service.head = &head{
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1}, [32]byte{2})
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, r1))
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)

	// Block in DB
	b = util.NewBeaconBlock()
	b.Block.Slot = 3
	util.SaveBlock(t, ctx, service.cfg.BeaconDB, b)
	r1, err = b.Block.HashTreeRoot()
	require.NoError(t, err)
	st, _ = util.DeterministicGenesisState(t, 1)
	service.head = &head{
		root:  r1,
		block: wsb,
		state: st,
	}
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(2, 1, [8]byte{1}, [32]byte{2})
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, r1))
	require.LogsDoNotContain(t, hook, invalidStateErr)
	require.LogsDoNotContain(t, hook, hookErr)
	vId, payloadID, has := service.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(2, [32]byte{2})
	require.Equal(t, true, has)
	require.Equal(t, primitives.ValidatorIndex(1), vId)
	require.Equal(t, [8]byte{1}, payloadID)

	// Test zero headRoot returns immediately.
	headRoot := service.headRoot()
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, [32]byte{}))
	require.Equal(t, service.headRoot(), headRoot)
}

func TestService_forkchoiceUpdateWithExecution_SameHeadRootNewProposer(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	altairBlk := util.SaveBlock(t, ctx, beaconDB, util.NewBeaconBlockAltair())
	altairBlkRoot, err := altairBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	bellatrixBlk := util.SaveBlock(t, ctx, beaconDB, util.NewBeaconBlockBellatrix())
	bellatrixBlkRoot, err := bellatrixBlk.Block().HashTreeRoot()
	require.NoError(t, err)
	fcs := doublylinkedtree.New()
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB, fcs)),
		WithForkChoiceStore(fcs),
		WithProposerIdsCache(cache.NewProposerPayloadIDsCache()),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	st, _ := util.DeterministicGenesisState(t, 10)
	service.head = &head{
		state: st,
	}

	ojc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	ofc := &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}
	state, blkRoot, err := prepareForkchoiceState(ctx, 0, [32]byte{}, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 1, altairBlkRoot, [32]byte{}, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))
	state, blkRoot, err = prepareForkchoiceState(ctx, 2, bellatrixBlkRoot, altairBlkRoot, params.BeaconConfig().ZeroHash, ojc, ofc)
	require.NoError(t, err)
	require.NoError(t, fcs.InsertNode(ctx, state, blkRoot))

	service.cfg.ExecutionEngineCaller = &mockExecution.EngineClient{}
	require.NoError(t, beaconDB.SaveState(ctx, st, bellatrixBlkRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, bellatrixBlkRoot))
	sb, err := blocks.NewSignedBeaconBlock(util.HydrateSignedBeaconBlockBellatrix(&ethpb.SignedBeaconBlockBellatrix{}))
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, sb))
	r, err := sb.Block().HashTreeRoot()
	require.NoError(t, err)

	// Set head to be the same but proposing next slot
	service.head.root = r
	service.cfg.ProposerSlotIndexCache.SetProposerAndPayloadIDs(service.CurrentSlot()+1, 0, [8]byte{}, [32]byte{} /* root */)
	require.NoError(t, service.forkchoiceUpdateWithExecution(ctx, r))

}
