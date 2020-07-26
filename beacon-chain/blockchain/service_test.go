package blockchain

import (
	"bytes"
	"context"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	protodb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

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

var _ = p2p.Broadcaster(&mockBroadcaster{})

func setupBeaconChain(t *testing.T, beaconDB db.Database, sc *cache.StateSummaryCache) *Service {
	endpoint := "http://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Service
	var err error
	bState, _ := testutil.DeterministicGenesisState(t, 10)
	err = beaconDB.SavePowchainData(ctx, &protodb.ETH1ChainData{
		BeaconState: bState.InnerStateUnsafe(),
		Trie:        &protodb.SparseMerkleTrie{},
		CurrentEth1Data: &protodb.LatestETH1Data{
			BlockHash: make([]byte, 32),
		},
		ChainstartData: &protodb.ChainStartData{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot:  make([]byte, 32),
				DepositCount: 0,
				BlockHash:    make([]byte, 32),
			},
		},
		DepositContainers: []*protodb.DepositContainer{},
	})
	require.NoError(t, err)
	web3Service, err = powchain.NewService(ctx, &powchain.Web3ServiceConfig{
		BeaconDB:        beaconDB,
		HTTPEndPoint:    endpoint,
		DepositContract: common.Address{},
	})
	require.NoError(t, err, "Unable to set up web3 service")

	opsService, err := attestations.NewService(ctx, &attestations.Config{Pool: attestations.NewPool()})
	require.NoError(t, err)

	depositCache, err := depositcache.NewDepositCache()
	require.NoError(t, err)

	cfg := &Config{
		BeaconBlockBuf:    0,
		BeaconDB:          beaconDB,
		DepositCache:      depositCache,
		ChainStartFetcher: web3Service,
		P2p:               &mockBroadcaster{},
		StateNotifier:     &mockBeaconNode{},
		AttPool:           attestations.NewPool(),
		StateGen:          stategen.New(beaconDB, sc),
		ForkChoiceStore:   protoarray.New(0, 0, params.BeaconConfig().ZeroHash),
		OpsService:        opsService,
	}

	chainService, err := NewService(ctx, cfg)
	require.NoError(t, err, "Unable to setup chain service")
	chainService.genesisTime = time.Unix(1, 0) // non-zero time

	return chainService
}

func TestChainStartStop_Uninitialized(t *testing.T) {
	hook := logTest.NewGlobal()
	db, sc := testDB.SetupDB(t)
	chainService := setupBeaconChain(t, db, sc)

	// Listen for state events.
	stateSubChannel := make(chan *feed.Event, 1)
	stateSub := chainService.stateNotifier.StateFeed().Subscribe(stateSubChannel)

	// Test the chain start state notifier.
	genesisTime := time.Unix(1, 0)
	chainService.Start()
	event := &feed.Event{
		Type: statefeed.ChainStarted,
		Data: &statefeed.ChainStartedData{
			StartTime: genesisTime,
		},
	}
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 1; sent == 1; {
		sent = chainService.stateNotifier.StateFeed().Send(event)
		if sent == 1 {
			// Flush our local subscriber.
			<-stateSubChannel
		}
	}

	// Now wait for notification the state is ready.
	for stateInitialized := false; stateInitialized == false; {
		recv := <-stateSubChannel
		if recv.Type == statefeed.Initialized {
			stateInitialized = true
		}
	}
	stateSub.Unsubscribe()

	beaconState, err := db.HeadState(context.Background())
	require.NoError(t, err)
	if beaconState == nil || beaconState.Slot() != 0 {
		t.Error("Expected canonical state feed to send a state with genesis block")
	}
	require.NoError(t, chainService.Stop(), "Unable to stop chain service")
	// The context should have been canceled.
	assert.Equal(t, context.Canceled, chainService.ctx.Err(), "Context was not canceled")
	testutil.AssertLogsContain(t, hook, "Waiting")
	testutil.AssertLogsContain(t, hook, "Initialized beacon chain genesis state")
}

func TestChainStartStop_Initialized(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)

	genesisBlk := testutil.NewBeaconBlock()
	blkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveBlock(ctx, genesisBlk))
	s := testutil.NewBeaconState()
	require.NoError(t, s.SetSlot(1))
	require.NoError(t, db.SaveState(ctx, s, blkRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blkRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blkRoot))
	require.NoError(t, db.SaveJustifiedCheckpoint(ctx, &ethpb.Checkpoint{Root: blkRoot[:]}))

	// Test the start function.
	chainService.Start()

	require.NoError(t, chainService.Stop(), "Unable to stop chain service")

	// The context should have been canceled.
	assert.Equal(t, context.Canceled, chainService.ctx.Err(), "Context was not canceled")
	testutil.AssertLogsContain(t, hook, "data already exists")
}

func TestChainService_InitializeBeaconChain(t *testing.T) {
	helpers.ClearCache()
	db, sc := testDB.SetupDB(t)
	ctx := context.Background()

	bc := setupBeaconChain(t, db, sc)
	var err error

	// Set up 10 deposits pre chain start for validators to register
	count := uint64(10)
	deposits, _, err := testutil.DeterministicDepositsAndKeys(count)
	require.NoError(t, err)
	trie, _, err := testutil.DepositTrieFromDeposits(deposits)
	require.NoError(t, err)
	hashTreeRoot := trie.HashTreeRoot()
	genState, err := state.EmptyGenesisState()
	require.NoError(t, err)
	err = genState.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot:  hashTreeRoot[:],
		DepositCount: uint64(len(deposits)),
	})
	genState, err = b.ProcessPreGenesisDeposits(ctx, genState, deposits)
	require.NoError(t, err)

	_, err = bc.initializeBeaconChain(ctx, time.Unix(0, 0), genState, &ethpb.Eth1Data{DepositRoot: hashTreeRoot[:]})
	require.NoError(t, err)

	_, err = bc.HeadState(ctx)
	assert.NoError(t, err)
	headBlk, err := bc.HeadBlock(ctx)
	require.NoError(t, err)
	if headBlk == nil {
		t.Error("Head state can't be nil after initialize beacon chain")
	}
	if bc.headRoot() == params.BeaconConfig().ZeroHash {
		t.Error("Canonical root for slot 0 can't be zeros after initialize beacon chain")
	}
}

func TestChainService_InitializeChainInfo(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	ctx := context.Background()

	genesis := testutil.NewBeaconBlock()
	genesisRoot, err := stateutil.BlockRoot(genesis.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, genesisRoot))
	require.NoError(t, db.SaveBlock(ctx, genesis))

	finalizedSlot := params.BeaconConfig().SlotsPerEpoch*2 + 1
	headBlock := testutil.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot
	headBlock.Block.ParentRoot = bytesutil.PadTo(genesisRoot[:], 32)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(finalizedSlot))
	require.NoError(t, headState.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]))
	headRoot, err := stateutil.BlockRoot(headBlock.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, headState, headRoot))
	require.NoError(t, db.SaveState(ctx, headState, genesisRoot))
	require.NoError(t, db.SaveBlock(ctx, headBlock))
	if err := db.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{
		Epoch: helpers.SlotToEpoch(finalizedSlot),
		Root:  headRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	c := &Service{beaconDB: db, stateGen: stategen.New(db, sc)}
	require.NoError(t, c.initializeChainInfo(ctx))
	headBlk, err := c.HeadBlock(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, headBlock, headBlk, "Head block incorrect")
	s, err := c.HeadState(ctx)
	require.NoError(t, err)
	assert.DeepEqual(t, headState.InnerStateUnsafe(), s.InnerStateUnsafe(), "Head state incorrect")
	assert.Equal(t, c.HeadSlot(), headBlock.Block.Slot, "Head slot incorrect")
	r, err := c.HeadRoot(context.Background())
	require.NoError(t, err)
	if !bytes.Equal(headRoot[:], r) {
		t.Error("head slot incorrect")
	}
	assert.Equal(t, genesisRoot, c.genesisRoot, "Genesis block root incorrect")
}

func TestChainService_SaveHeadNoDB(t *testing.T) {
	db, sc := testDB.SetupDB(t)
	ctx := context.Background()
	s := &Service{
		beaconDB: db,
		stateGen: stategen.New(db, sc),
	}
	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	r, err := ssz.HashTreeRoot(b)
	require.NoError(t, err)
	newState := testutil.NewBeaconState()
	require.NoError(t, s.stateGen.SaveState(ctx, r, newState))
	require.NoError(t, s.saveHeadNoDB(ctx, b, r, newState))

	newB, err := s.beaconDB.HeadBlock(ctx)
	require.NoError(t, err)
	if reflect.DeepEqual(newB, b) {
		t.Error("head block should not be equal")
	}
}

func TestHasBlock_ForkChoiceAndDB(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)
	s := &Service{
		forkChoiceStore:  protoarray.New(0, 0, [32]byte{}),
		finalizedCheckpt: &ethpb.Checkpoint{},
		beaconDB:         db,
	}
	block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}}
	r, err := stateutil.BlockRoot(block.Block)
	require.NoError(t, err)
	bs := &pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{}, CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}
	state, err := beaconstate.InitializeFromProto(bs)
	require.NoError(t, err)
	require.NoError(t, s.insertBlockAndAttestationsToForkChoiceStore(ctx, block.Block, r, state))

	assert.Equal(t, false, s.hasBlock(ctx, [32]byte{}), "Should not have block")
	assert.Equal(t, true, s.hasBlock(ctx, r), "Should have block")
}

func BenchmarkHasBlockDB(b *testing.B) {
	db, _ := testDB.SetupDB(b)
	ctx := context.Background()
	s := &Service{
		beaconDB: db,
	}
	block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	require.NoError(b, s.beaconDB.SaveBlock(ctx, block))
	r, err := stateutil.BlockRoot(block.Block)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, s.beaconDB.HasBlock(ctx, r), "Block is not in DB")
	}
}

func BenchmarkHasBlockForkChoiceStore(b *testing.B) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(b)
	s := &Service{
		forkChoiceStore:  protoarray.New(0, 0, [32]byte{}),
		finalizedCheckpt: &ethpb.Checkpoint{},
		beaconDB:         db,
	}
	block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Body: &ethpb.BeaconBlockBody{}}}
	r, err := stateutil.BlockRoot(block.Block)
	require.NoError(b, err)
	bs := &pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{}, CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}
	state, err := beaconstate.InitializeFromProto(bs)
	require.NoError(b, err)
	require.NoError(b, s.insertBlockAndAttestationsToForkChoiceStore(ctx, block.Block, r, state))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		require.Equal(b, true, s.forkChoiceStore.HasNode(r), "Block is not in fork choice store")
	}
}
