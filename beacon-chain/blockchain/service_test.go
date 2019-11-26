package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	ssz "github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/statefeed"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
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

type store struct {
	headRoot []byte
}

func (s *store) OnBlock(ctx context.Context, b *ethpb.BeaconBlock) error {
	return nil
}

func (s *store) OnBlockInitialSyncStateTransition(ctx context.Context, b *ethpb.BeaconBlock) error {
	return nil
}

func (s *store) OnAttestation(ctx context.Context, a *ethpb.Attestation) error {
	return nil
}

func (s *store) GenesisStore(ctx context.Context, justifiedCheckpoint *ethpb.Checkpoint, finalizedCheckpoint *ethpb.Checkpoint) error {
	return nil
}

func (s *store) FinalizedCheckpt() *ethpb.Checkpoint {
	return nil
}

func (s *store) Head(ctx context.Context) ([]byte, error) {
	return s.headRoot, nil
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

type mockOperationService struct{}

func (ms *mockOperationService) IncomingProcessedBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return nil
}

func (ms *mockOperationService) AttestationPool(ctx context.Context, requestedSlot uint64) ([]*ethpb.Attestation, error) {
	return nil, nil
}

func (ms *mockOperationService) AttestationPoolNoVerify(ctx context.Context) ([]*ethpb.Attestation, error) {
	return nil, nil
}

func (ms *mockOperationService) AttestationPoolForForkchoice(ctx context.Context) ([]*ethpb.Attestation, error) {
	return nil, nil
}

func (ms *mockOperationService) AttestationsBySlotCommittee(ctx context.Context, slot uint64, index uint64) ([]*ethpb.Attestation, error) {
	return nil, nil
}

type mockClient struct{}

func (m *mockClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (m *mockClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	head := &gethTypes.Header{Number: big.NewInt(0), Difficulty: big.NewInt(100)}
	return gethTypes.NewBlockWithHeader(head), nil
}

func (m *mockClient) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	head := &gethTypes.Header{Number: big.NewInt(0), Difficulty: big.NewInt(100)}
	return gethTypes.NewBlockWithHeader(head), nil
}

func (m *mockClient) HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error) {
	return &gethTypes.Header{Number: big.NewInt(0), Difficulty: big.NewInt(100)}, nil
}

func (m *mockClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (m *mockClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return []byte{'t', 'e', 's', 't'}, nil
}

func (m *mockClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	return []byte{'t', 'e', 's', 't'}, nil
}

func (m *mockClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	logs := make([]gethTypes.Log, 3)
	for i := 0; i < len(logs); i++ {
		logs[i].Address = common.Address{}
		logs[i].Topics = make([]common.Hash, 5)
		logs[i].Topics[0] = common.Hash{'a'}
		logs[i].Topics[1] = common.Hash{'b'}
		logs[i].Topics[2] = common.Hash{'c'}

	}
	return logs, nil
}

func (m *mockClient) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{'A'})
}

type faultyClient struct{}

func (f *faultyClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *faultyClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	return nil, errors.New("failed")
}

func (f *faultyClient) BlockByNumber(ctx context.Context, number *big.Int) (*gethTypes.Block, error) {
	return nil, errors.New("failed")
}

func (f *faultyClient) HeaderByNumber(ctx context.Context, number *big.Int) (*gethTypes.Header, error) {
	return nil, errors.New("failed")
}

func (f *faultyClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *faultyClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	return nil, errors.New("unable to retrieve logs")
}

func (f *faultyClient) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return []byte{}, errors.New("unable to retrieve contract code")
}

func (f *faultyClient) CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error) {
	return []byte{}, errors.New("unable to retrieve contract code")
}

func (f *faultyClient) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{'A'})
}

type mockBroadcaster struct {
	broadcastCalled bool
}

func (mb *mockBroadcaster) Broadcast(_ context.Context, _ proto.Message) error {
	mb.broadcastCalled = true
	return nil
}

var _ = p2p.Broadcaster(&mockBroadcaster{})

func setupGenesisBlock(t *testing.T, cs *Service) ([32]byte, *ethpb.BeaconBlock) {
	genesis := b.NewGenesisBlock([]byte{})
	if err := cs.beaconDB.SaveBlock(context.Background(), genesis); err != nil {
		t.Fatalf("could not save block to db: %v", err)
	}
	parentHash, err := ssz.SigningRoot(genesis)
	if err != nil {
		t.Fatalf("unable to get tree hash root of canonical head: %v", err)
	}
	return parentHash, genesis
}

func setupBeaconChain(t *testing.T, beaconDB db.Database) *Service {
	endpoint := "ws://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Service
	var err error
	web3Service, err = powchain.NewService(ctx, &powchain.Web3ServiceConfig{
		ETH1Endpoint:    endpoint,
		DepositContract: common.Address{},
	})
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf:    0,
		BeaconDB:          beaconDB,
		DepositCache:      depositcache.NewDepositCache(),
		ChainStartFetcher: web3Service,
		OpsPoolService:    &mockOperationService{},
		P2p:               &mockBroadcaster{},
		StateNotifier:     &mockBeaconNode{},
	}
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	return chainService
}

func TestChainStartStop_Uninitialized(t *testing.T) {
	helpers.ClearAllCaches()
	hook := logTest.NewGlobal()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	chainService := setupBeaconChain(t, db)

	// Listen for state events.
	stateSubChannel := make(chan *statefeed.Event, 1)
	stateSub := chainService.stateNotifier.StateFeed().Subscribe(stateSubChannel)

	// Test the chain start state notifier.
	genesisTime := time.Unix(0, 0)
	chainService.Start()
	event := &statefeed.Event{
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
		if recv.Type == statefeed.StateInitialized {
			stateInitialized = true
		}
	}
	stateSub.Unsubscribe()

	beaconState, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if beaconState == nil || beaconState.Slot != 0 {
		t.Error("Expected canonical state feed to send a state with genesis block")
	}
	if err := chainService.Stop(); err != nil {
		t.Fatalf("Unable to stop chain service: %v", err)
	}
	// The context should have been canceled.
	if chainService.ctx.Err() != context.Canceled {
		t.Error("Context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, "Waiting")
	testutil.AssertLogsContain(t, hook, "Genesis time reached")
}

func TestChainStartStop_Initialized(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	chainService := setupBeaconChain(t, db)

	genesisBlk := b.NewGenesisBlock([]byte{})
	blkRoot, err := ssz.SigningRoot(genesisBlk)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, genesisBlk); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(ctx, blkRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(ctx, &pb.BeaconState{Slot: 1}, blkRoot); err != nil {
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
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	bc := setupBeaconChain(t, db)

	// Set up 10 deposits pre chain start for validators to register
	count := uint64(10)
	deposits, _, _ := testutil.SetupInitialDeposits(t, count)
	if err := bc.initializeBeaconChain(ctx, time.Unix(0, 0), deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatal(err)
	}

	s, err := bc.beaconDB.State(ctx, bytesutil.ToBytes32(bc.canonicalRoots[0]))
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range s.Validators {
		if !db.HasValidatorIndex(ctx, bytesutil.ToBytes48(v.PublicKey)) {
			t.Errorf("Validator %s missing from db", hex.EncodeToString(v.PublicKey))
		}
	}

	if _, err := bc.HeadState(ctx); err != nil {
		t.Error(err)
	}
	if bc.HeadBlock() == nil {
		t.Error("Head state can't be nil after initialize beacon chain")
	}
	if bc.CanonicalRoot(0) == nil {
		t.Error("Canonical root for slot 0 can't be nil after initialize beacon chain")
	}
}

func TestChainService_InitializeChainInfo(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	genesis := b.NewGenesisBlock([]byte{})
	genesisRoot, err := ssz.SigningRoot(genesis)
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
	headBlock := &ethpb.BeaconBlock{Slot: finalizedSlot, ParentRoot: genesisRoot[:]}
	headState := &pb.BeaconState{Slot: finalizedSlot}
	headRoot, _ := ssz.SigningRoot(headBlock)
	if err := db.SaveState(ctx, headState, headRoot); err != nil {
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
	if err := db.SaveBlock(ctx, headBlock); err != nil {
		t.Fatal(err)
	}
	c := &Service{beaconDB: db, canonicalRoots: make(map[uint64][]byte)}
	if err := c.initializeChainInfo(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.HeadBlock(), headBlock) {
		t.Error("head block incorrect")
	}
	s, err := c.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, headState) {
		t.Error("head state incorrect")
	}
	if headBlock.Slot != c.HeadSlot() {
		t.Error("head slot incorrect")
	}
	if !bytes.Equal(headRoot[:], c.HeadRoot()) {
		t.Error("head slot incorrect")
	}
}
