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
	if err != nil {
		t.Fatal(err)
	}
	web3Service, err = powchain.NewService(ctx, &powchain.Web3ServiceConfig{
		BeaconDB:        beaconDB,
		HTTPEndPoint:    endpoint,
		DepositContract: common.Address{},
	})
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}

	opsService, err := attestations.NewService(ctx, &attestations.Config{Pool: attestations.NewPool()})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		BeaconBlockBuf:    0,
		BeaconDB:          beaconDB,
		DepositCache:      depositcache.NewDepositCache(),
		ChainStartFetcher: web3Service,
		P2p:               &mockBroadcaster{},
		StateNotifier:     &mockBeaconNode{},
		AttPool:           attestations.NewPool(),
		StateGen:          stategen.New(beaconDB, sc),
		ForkChoiceStore:   protoarray.New(0, 0, params.BeaconConfig().ZeroHash),
		OpsService:        opsService,
	}

	// Safe a state in stategen to purposes of testing a service stop / shutdown.
	if err := cfg.StateGen.SaveState(ctx, bytesutil.ToBytes32(bState.FinalizedCheckpoint().Root), bState); err != nil {
		t.Fatal(err)
	}

	chainService, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}
	chainService.genesisTime = time.Unix(1, 0) // non-zero time

	return chainService
}

func TestChainStartStop_Initialized(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	chainService := setupBeaconChain(t, db, sc)

	genesisBlk := testutil.NewBeaconBlock()
	blkRoot, err := stateutil.BlockRoot(genesisBlk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, genesisBlk); err != nil {
		t.Fatal(err)
	}
	s := testutil.NewBeaconState()
	if err := s.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, s, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := chainService.stateGen.SaveState(ctx, bytesutil.ToBytes32(s.FinalizedCheckpoint().Root), s); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveJustifiedCheckpoint(ctx, &ethpb.Checkpoint{Root: blkRoot[:]}); err != nil {
		t.Fatal(err)
	}

	// Test the start function.
	chainService.Start()

	if err := chainService.Stop(); err != nil {
		t.Fatalf("unable to stop chain service: %v", err)
	}

	// The context should have been canceled.
	if chainService.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	trie, _, err := testutil.DepositTrieFromDeposits(deposits)
	if err != nil {
		t.Fatal(err)
	}
	hashTreeRoot := trie.HashTreeRoot()
	genState, err := state.EmptyGenesisState()
	if err != nil {
		t.Fatal(err)
	}
	err = genState.SetEth1Data(&ethpb.Eth1Data{
		DepositRoot:  hashTreeRoot[:],
		DepositCount: uint64(len(deposits)),
	})
	genState, err = b.ProcessPreGenesisDeposits(ctx, genState, deposits)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := bc.initializeBeaconChain(ctx, time.Unix(0, 0), genState, &ethpb.Eth1Data{
		DepositRoot: hashTreeRoot[:],
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := bc.HeadState(ctx); err != nil {
		t.Error(err)
	}
	headBlk, err := bc.HeadBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, genesisRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, genesis); err != nil {
		t.Fatal(err)
	}

	finalizedSlot := params.BeaconConfig().SlotsPerEpoch*2 + 1
	headBlock := testutil.NewBeaconBlock()
	headBlock.Block.Slot = finalizedSlot
	headBlock.Block.ParentRoot = bytesutil.PadTo(genesisRoot[:], 32)
	headState := testutil.NewBeaconState()
	if err := headState.SetSlot(finalizedSlot); err != nil {
		t.Fatal(err)
	}
	if err := headState.SetGenesisValidatorRoot(params.BeaconConfig().ZeroHash[:]); err != nil {
		t.Fatal(err)
	}
	headRoot, err := stateutil.BlockRoot(headBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, headState, headRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, headState, genesisRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, headBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveFinalizedCheckpoint(ctx, &ethpb.Checkpoint{
		Epoch: helpers.SlotToEpoch(finalizedSlot),
		Root:  headRoot[:],
	}); err != nil {
		t.Fatal(err)
	}
	c := &Service{beaconDB: db, stateGen: stategen.New(db, sc)}
	if err := c.initializeChainInfo(ctx); err != nil {
		t.Fatal(err)
	}
	headBlk, err := c.HeadBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headBlk, headBlock) {
		t.Error("head block incorrect")
	}
	s, err := c.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.InnerStateUnsafe(), headState.InnerStateUnsafe()) {
		t.Error("head state incorrect")
	}
	if headBlock.Block.Slot != c.HeadSlot() {
		t.Error("head slot incorrect")
	}
	r, err := c.HeadRoot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(headRoot[:], r) {
		t.Error("head slot incorrect")
	}
	if c.genesisRoot != genesisRoot {
		t.Error("genesis block root incorrect")
	}
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
	if err != nil {
		t.Fatal(err)
	}
	newState := testutil.NewBeaconState()
	if err := s.stateGen.SaveState(ctx, r, newState); err != nil {
		t.Fatal(err)
	}

	if err := s.saveHeadNoDB(ctx, b, r); err != nil {
		t.Fatal(err)
	}

	newB, err := s.beaconDB.HeadBlock(ctx)
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	bs := &pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{}, CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}
	state, err := beaconstate.InitializeFromProto(bs)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.insertBlockToForkChoiceStore(ctx, block.Block, r, state); err != nil {
		t.Fatal(err)
	}

	if s.hasBlock(ctx, [32]byte{}) {
		t.Error("Should not have block")
	}

	if !s.hasBlock(ctx, r) {
		t.Error("Should have block")
	}
}

func BenchmarkHasBlockDB(b *testing.B) {
	db, _ := testDB.SetupDB(b)
	ctx := context.Background()
	s := &Service{
		beaconDB: db,
	}
	block := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := s.beaconDB.SaveBlock(ctx, block); err != nil {
		b.Fatal(err)
	}
	r, err := stateutil.BlockRoot(block.Block)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !s.beaconDB.HasBlock(ctx, r) {
			b.Fatal("Block is not in DB")
		}
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
	if err != nil {
		b.Fatal(err)
	}
	bs := &pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{}, CurrentJustifiedCheckpoint: &ethpb.Checkpoint{}}
	state, err := beaconstate.InitializeFromProto(bs)
	if err != nil {
		b.Fatal(err)
	}
	if err := s.insertBlockToForkChoiceStore(ctx, block.Block, r, state); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !s.forkChoiceStore.HasNode(r) {
			b.Fatal("Block is not in fork choice store")
		}
	}
}
