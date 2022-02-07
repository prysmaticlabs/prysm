package blockchain

import (
	"bytes"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain/store"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"google.golang.org/protobuf/proto"
)

type mockBeaconNode struct {
	stateFeed *event.Feed
}

// StateFeed mocks the same method in the beacon node.
func (mbn *mockBeaconNode) StateFeed() *event.Feed {
	if mbn.stateFeed == nil {
		mbn.stateFeed = new(event.Feed)
	}
	return mbn.stateFeed
}

type mockBroadcaster struct {
	broadcastCalled bool
}

func (mb *mockBroadcaster) Broadcast(_ context.Context, _ proto.Message) error {
	mb.broadcastCalled = true
	return nil
}

func (mb *mockBroadcaster) BroadcastAttestation(_ context.Context, _ uint64, _ *ethpb.Attestation) error {
	mb.broadcastCalled = true
	return nil
}

func (mb *mockBroadcaster) BroadcastSyncCommitteeMessage(_ context.Context, _ uint64, _ *ethpb.SyncCommitteeMessage) error {
	mb.broadcastCalled = true
	return nil
}

var _ p2p.Broadcaster = (*mockBroadcaster)(nil)

func setupBeaconChain(t *testing.T, beaconDB db.Database) *Service {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Service
	var err error
	bState, _ := util.DeterministicGenesisState(t, 10)
	pbState, err := v1.ProtobufBeaconState(bState.InnerStateUnsafe())
	require.NoError(t, err)
	err = beaconDB.SavePowchainData(ctx, &ethpb.ETH1ChainData{
		BeaconState: pbState,
		Trie:        &ethpb.SparseMerkleTrie{},
		CurrentEth1Data: &ethpb.LatestETH1Data{
			BlockHash: make([]byte, 32),
		},
		ChainstartData: &ethpb.ChainStartData{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot:  make([]byte, 32),
				DepositCount: 0,
				BlockHash:    make([]byte, 32),
			},
		},
		DepositContainers: []*ethpb.DepositContainer{},
	})
	require.NoError(t, err)
	web3Service, err = powchain.NewService(
		ctx,
		powchain.WithDatabase(beaconDB),
		powchain.WithHttpEndpoints([]string{endpoint}),
		powchain.WithDepositContractAddress(common.Address{}),
	)
	require.NoError(t, err, "Unable to set up web3 service")

	attService, err := attestations.NewService(ctx, &attestations.Config{Pool: attestations.NewPool()})
	require.NoError(t, err)

	depositCache, err := depositcache.New()
	require.NoError(t, err)

	stateGen := stategen.New(beaconDB)
	// Safe a state in stategen to purposes of testing a service stop / shutdown.
	require.NoError(t, stateGen.SaveState(ctx, bytesutil.ToBytes32(bState.FinalizedCheckpoint().Root), bState))

	opts := []Option{
		WithDatabase(beaconDB),
		WithDepositCache(depositCache),
		WithChainStartFetcher(web3Service),
		WithAttestationPool(attestations.NewPool()),
		WithP2PBroadcaster(&mockBroadcaster{}),
		WithStateNotifier(&mockBeaconNode{}),
		WithForkChoiceStore(protoarray.New(0, 0, params.BeaconConfig().ZeroHash)),
		WithAttestationService(attService),
		WithStateGen(stateGen),
	}

	chainService, err := NewService(ctx, opts...)
	require.NoError(t, err, "Unable to setup chain service")
	chainService.genesisTime = time.Unix(1, 0) // non-zero time

	return chainService
}

func TestChainStartStop_Initialized(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, beaconDB)

	genesisBlk := util.NewBeaconBlock()
	blkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlk)))
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, beaconDB.SaveState(ctx, s, blkRoot))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, blkRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, beaconDB.SaveJustifiedCheckpoint(ctx, &ethpb.Checkpoint{Root: blkRoot[:]}))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Root: blkRoot[:]}))
	chainService.cfg.FinalizedStateAtStartUp = s
	// Test the start function.
	chainService.Start()

	require.NoError(t, chainService.Stop(), "Unable to stop chain service")

	// The context should have been canceled.
	assert.Equal(t, context.Canceled, chainService.ctx.Err(), "Context was not canceled")
	require.LogsContain(t, hook, "data already exists")
}

func TestChainStartStop_GenesisZeroHashes(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, beaconDB)

	genesisBlk := util.NewBeaconBlock()
	blkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlk)))
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, s, blkRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, beaconDB.SaveJustifiedCheckpoint(ctx, &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]}))
	chainService.cfg.FinalizedStateAtStartUp = s
	// Test the start function.
	chainService.Start()

	require.NoError(t, chainService.Stop(), "Unable to stop chain service")

	// The context should have been canceled.
	assert.Equal(t, context.Canceled, chainService.ctx.Err(), "Context was not canceled")
	require.LogsContain(t, hook, "data already exists")
}

func TestChainService_InitializeBeaconChain(t *testing.T) {
	helpers.ClearCache()
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()

	bc := setupBeaconChain(t, beaconDB)
	var err error

	// Set up 10 deposits pre chain start for validators to register
	count := uint64(10)
	deposits, _, err := util.DeterministicDepositsAndKeys(count)
	require.NoError(t, err)
	trie, _, err := util.DepositTrieFromDeposits(deposits)
	require.NoError(t, err)
	hashTreeRoot := trie.HashTreeRoot()
	genState, err := transition.EmptyGenesisState()
	require.NoError(t, err)
	err = genState.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot:  hashTreeRoot[:],
		DepositCount: uint64(len(deposits)),
		BlockHash:    make([]byte, 32),
	})
	require.NoError(t, err)
	genState, err = b.ProcessPreGenesisDeposits(ctx, genState, deposits)
	require.NoError(t, err)

	_, err = bc.initializeBeaconChain(ctx, time.Unix(0, 0), genState, &ethpb.Eth1Data{DepositRoot: hashTreeRoot[:], BlockHash: make([]byte, 32)})
	require.NoError(t, err)

	_, err = bc.HeadState(ctx)
	assert.NoError(t, err)
	headBlk, err := bc.HeadBlock(ctx)
	require.NoError(t, err)
	if headBlk == nil {
		t.Error("Head state can't be nil after initialize beacon chain")
	}
	r, err := bc.HeadRoot(ctx)
	require.NoError(t, err)
	if bytesutil.ToBytes32(r) == params.BeaconConfig().ZeroHash {
		t.Error("Canonical root for slot 0 can't be zeros after initialize beacon chain")
	}
}

func TestChainService_CorrectGenesisRoots(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, beaconDB)

	genesisBlk := util.NewBeaconBlock()
	blkRoot, err := genesisBlk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlk)))
	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(0))
	require.NoError(t, beaconDB.SaveState(ctx, s, blkRoot))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, blkRoot))
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Root: blkRoot[:]}))
	chainService.cfg.FinalizedStateAtStartUp = s
	// Test the start function.
	chainService.Start()

	require.DeepEqual(t, blkRoot[:], chainService.store.FinalizedCheckpt().Root, "Finalize Checkpoint root is incorrect")
	require.DeepEqual(t, params.BeaconConfig().ZeroHash[:], chainService.store.JustifiedCheckpt().Root, "Justified Checkpoint root is incorrect")

	require.NoError(t, chainService.Stop(), "Unable to stop chain service")

}

func TestChainService_InitializeChainInfo(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()

	genesis := util.NewBeaconBlock()
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))

	finalizedSlot := params.BeaconConfig().SlotsPerEpoch*2 + 1
	headBlock := util.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot
	headBlock.Block.ParentRoot = bytesutil.PadTo(genesisRoot[:], 32)
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(finalizedSlot))
	require.NoError(t, headState.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, headState, headRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, genesisRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlock)))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{Epoch: slots.ToEpoch(finalizedSlot), Root: headRoot[:]}))
	attSrv, err := attestations.NewService(ctx, &attestations.Config{})
	require.NoError(t, err)
	c, err := NewService(ctx, WithDatabase(beaconDB), WithStateGen(stategen.New(beaconDB)), WithAttestationService(attSrv), WithStateNotifier(&mock.MockStateNotifier{}), WithFinalizedStateAtStartUp(headState))
	require.NoError(t, err)
	require.NoError(t, c.startFromSavedState(headState))
	headBlk, err := c.HeadBlock(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, headBlock, headBlk.Proto(), "Head block incorrect")
	s, err := c.HeadState(ctx)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Head state incorrect")
	assert.Equal(t, c.HeadSlot(), headBlock.Block.Slot, "Head slot incorrect")
	r, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	if !bytes.Equal(headRoot[:], r) {
		t.Error("head slot incorrect")
	}
	assert.Equal(t, genesisRoot, c.originBlockRoot, "Genesis block root incorrect")
}

func TestChainService_InitializeChainInfo_SetHeadAtGenesis(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()

	genesis := util.NewBeaconBlock()
	genesisRoot, err := genesis.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesis)))

	finalizedSlot := params.BeaconConfig().SlotsPerEpoch*2 + 1
	headBlock := util.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot
	headBlock.Block.ParentRoot = bytesutil.PadTo(genesisRoot[:], 32)
	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(finalizedSlot))
	require.NoError(t, headState.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveState(ctx, headState, headRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, genesisRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlock)))
	attSrv, err := attestations.NewService(ctx, &attestations.Config{})
	require.NoError(t, err)
	c, err := NewService(ctx, WithDatabase(beaconDB), WithStateGen(stategen.New(beaconDB)), WithAttestationService(attSrv), WithStateNotifier(&mock.MockStateNotifier{}))
	require.NoError(t, err)
	require.NoError(t, c.startFromSavedState(headState))
	s, err := c.HeadState(ctx)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Head state incorrect")
	assert.Equal(t, genesisRoot, c.originBlockRoot, "Genesis block root incorrect")
	assert.DeepEqual(t, genesis, c.head.block.Proto())
}

func TestChainService_InitializeChainInfo_HeadSync(t *testing.T) {
	resetFlags := flags.Get()
	flags.Init(&flags.GlobalFlags{
		HeadSync: true,
	})
	defer func() {
		flags.Init(resetFlags)
	}()

	hook := logTest.NewGlobal()
	finalizedSlot := params.BeaconConfig().SlotsPerEpoch*2 + 1
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()

	genesisBlock := util.NewBeaconBlock()
	genesisRoot, err := genesisBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(genesisBlock)))

	finalizedBlock := util.NewBeaconBlock()
	finalizedBlock.Block.Slot = finalizedSlot
	finalizedBlock.Block.ParentRoot = genesisRoot[:]
	finalizedRoot, err := finalizedBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(finalizedBlock)))

	// Set head slot close to the finalization point, no head sync is triggered.
	headBlock := util.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot + params.BeaconConfig().SlotsPerEpoch*5
	headBlock.Block.ParentRoot = finalizedRoot[:]
	headRoot, err := headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlock)))

	headState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, headState.SetSlot(headBlock.Block.Slot))
	require.NoError(t, headState.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	require.NoError(t, beaconDB.SaveState(ctx, headState, genesisRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, finalizedRoot))
	require.NoError(t, beaconDB.SaveState(ctx, headState, headRoot))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))
	require.NoError(t, beaconDB.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{
		Epoch: slots.ToEpoch(finalizedBlock.Block.Slot),
		Root:  finalizedRoot[:],
	}))

	attSrv, err := attestations.NewService(ctx, &attestations.Config{})
	require.NoError(t, err)
	c, err := NewService(ctx, WithDatabase(beaconDB), WithStateGen(stategen.New(beaconDB)), WithAttestationService(attSrv), WithStateNotifier(&mock.MockStateNotifier{}), WithFinalizedStateAtStartUp(headState))
	require.NoError(t, err)
	require.NoError(t, c.startFromSavedState(headState))
	s, err := c.HeadState(ctx)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Head state incorrect")
	assert.Equal(t, genesisRoot, c.originBlockRoot, "Genesis block root incorrect")
	// Since head sync is not triggered, chain is initialized to the last finalization checkpoint.
	assert.DeepEqual(t, finalizedBlock, c.head.block.Proto())
	assert.LogsContain(t, hook, "resetting head from the checkpoint ('--head-sync' flag is ignored)")
	assert.LogsDoNotContain(t, hook, "Regenerating state from the last checkpoint at slot")

	// Set head slot far beyond the finalization point, head sync should be triggered.
	headBlock = util.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot + params.BeaconConfig().SlotsPerEpoch*headSyncMinEpochsAfterCheckpoint
	headBlock.Block.ParentRoot = finalizedRoot[:]
	headRoot, err = headBlock.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(headBlock)))
	require.NoError(t, beaconDB.SaveState(ctx, headState, headRoot))
	require.NoError(t, beaconDB.SaveHeadBlockRoot(ctx, headRoot))

	hook.Reset()
	require.NoError(t, c.initializeHeadFromDB(ctx))
	s, err = c.HeadState(ctx)
	require.NoError(t, err)
	assert.DeepSSZEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Head state incorrect")
	assert.Equal(t, genesisRoot, c.originBlockRoot, "Genesis block root incorrect")
	// Head slot is far beyond the latest finalized checkpoint, head sync is triggered.
	assert.DeepEqual(t, headBlock, c.head.block.Proto())
	assert.LogsContain(t, hook, "Regenerating state from the last checkpoint at slot 225")
	assert.LogsDoNotContain(t, hook, "resetting head from the checkpoint ('--head-sync' flag is ignored)")
}

func TestChainService_SaveHeadNoDB(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	ctx := context.Background()
	s := &Service{
		cfg: &config{BeaconDB: beaconDB, StateGen: stategen.New(beaconDB)},
	}
	blk := util.NewBeaconBlock()
	blk.Block.Slot = 1
	r, err := blk.HashTreeRoot()
	require.NoError(t, err)
	newState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.cfg.StateGen.SaveState(ctx, r, newState))
	require.NoError(t, s.saveHeadNoDB(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk), r, newState))

	newB, err := s.cfg.BeaconDB.HeadBlock(ctx)
	require.NoError(t, err)
	if reflect.DeepEqual(newB, blk) {
		t.Error("head block should not be equal")
	}
}

func TestHasBlock_ForkChoiceAndDB(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)
	s := &Service{
		cfg:   &config{ForkChoiceStore: protoarray.New(0, 0, [32]byte{}), BeaconDB: beaconDB},
		store: &store.Store{},
	}
	s.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]})
	block := util.NewBeaconBlock()
	r, err := block.Block.HashTreeRoot()
	require.NoError(t, err)
	beaconState, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.insertBlockAndAttestationsToForkChoiceStore(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), r, beaconState))

	assert.Equal(t, false, s.hasBlock(ctx, [32]byte{}), "Should not have block")
	assert.Equal(t, true, s.hasBlock(ctx, r), "Should have block")
}

func TestServiceStop_SaveCachedBlocks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	beaconDB := testDB.SetupDB(t)
	s := &Service{
		cfg:            &config{BeaconDB: beaconDB, StateGen: stategen.New(beaconDB)},
		ctx:            ctx,
		cancel:         cancel,
		initSyncBlocks: make(map[[32]byte]block.SignedBeaconBlock),
	}
	block := util.NewBeaconBlock()
	r, err := block.Block.HashTreeRoot()
	require.NoError(t, err)
	s.saveInitSyncBlock(r, wrapper.WrappedPhase0SignedBeaconBlock(block))
	require.NoError(t, s.Stop())
	require.Equal(t, true, s.cfg.BeaconDB.HasBlock(ctx, r))
}

func TestProcessChainStartTime_ReceivedFeed(t *testing.T) {
	beaconDB := testDB.SetupDB(t)
	service := setupBeaconChain(t, beaconDB)
	stateChannel := make(chan *feed.Event, 1)
	stateSub := service.cfg.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()
	service.onPowchainStart(context.Background(), time.Now())

	stateEvent := <-stateChannel
	require.Equal(t, int(stateEvent.Type), statefeed.Initialized)
	_, ok := stateEvent.Data.(*statefeed.InitializedData)
	require.Equal(t, true, ok)
}

func BenchmarkHasBlockDB(b *testing.B) {
	beaconDB := testDB.SetupDB(b)
	ctx := context.Background()
	s := &Service{
		cfg: &config{BeaconDB: beaconDB},
	}
	block := util.NewBeaconBlock()
	require.NoError(b, s.cfg.BeaconDB.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block)))
	r, err := block.Block.HashTreeRoot()
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, s.cfg.BeaconDB.HasBlock(ctx, r), "Block is not in DB")
	}
}

func BenchmarkHasBlockForkChoiceStore(b *testing.B) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(b)
	s := &Service{
		cfg:   &config{ForkChoiceStore: protoarray.New(0, 0, [32]byte{}), BeaconDB: beaconDB},
		store: &store.Store{},
	}
	s.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 0, Root: params.BeaconConfig().ZeroHash[:]})
	block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}}
	r, err := block.Block.HashTreeRoot()
	require.NoError(b, err)
	bs := &ethpb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)}, CurrentJustifiedCheckpoint: &ethpb.Checkpoint{Root: make([]byte, 32)}}
	beaconState, err := v1.InitializeFromProto(bs)
	require.NoError(b, err)
	require.NoError(b, s.insertBlockAndAttestationsToForkChoiceStore(ctx, wrapper.WrappedPhase0SignedBeaconBlock(block).Block(), r, beaconState))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, s.cfg.ForkChoiceStore.HasNode(r), "Block is not in fork choice store")
	}
}
