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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure Service implements interfaces.
var _ = ChainFeeds(&Service{})
var _ = NewHeadNotifier(&Service{})

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

func (s *store) OnAttestation(ctx context.Context, a *ethpb.Attestation) (uint64, error) {
	return 0, nil
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

type mockOperationService struct{}

func (ms *mockOperationService) IncomingProcessedBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return nil
}

func (ms *mockOperationService) IncomingExitFeed() *event.Feed {
	return nil
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
	client := &mockClient{}
	web3Service, err = powchain.NewService(ctx, &powchain.Web3ServiceConfig{
		Endpoint:        endpoint,
		DepositContract: common.Address{},
		Reader:          client,
		Client:          client,
		Logger:          client,
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

	// Test the start function.
	genesisChan := make(chan time.Time, 0)
	sub := chainService.stateInitializedFeed.Subscribe(genesisChan)
	defer sub.Unsubscribe()
	chainService.Start()
	chainService.chainStartChan <- time.Unix(0, 0)
	genesisTime := <-genesisChan
	if genesisTime != time.Unix(0, 0) {
		t.Errorf(
			"Expected genesis time to equal chainstart time (%v), received %v",
			time.Unix(0, 0),
			genesisTime,
		)
	}

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
	testutil.AssertLogsContain(t, hook, "Waiting for ChainStart log from the Validator Deposit Contract to start the beacon chain...")
	testutil.AssertLogsContain(t, hook, "ChainStart time reached, starting the beacon chain!")
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
	testutil.AssertLogsContain(t, hook, "Beacon chain data already exists, starting service")
}

func TestChainService_InitializeBeaconChain(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	ctx := context.Background()

	bc := setupBeaconChain(t, db)

	// Set up 10 deposits pre chain start for validators to register
	count := uint64(10)
	deposits, _ := testutil.SetupInitialDeposits(t, count)
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

	if bc.HeadState() == nil {
		t.Error("Head state can't be nil after initialize beacon chain")
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

	headBlock := &ethpb.BeaconBlock{Slot: 1}
	headState := &pb.BeaconState{Slot: 1}
	headRoot, _ := ssz.SigningRoot(headBlock)
	if err := db.SaveState(ctx, headState, headRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(ctx, headBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveHeadBlockRoot(ctx, headRoot); err != nil {
		t.Fatal(err)
	}
	c := &Service{beaconDB: db, canonicalRoots: make(map[uint64][]byte)}
	if err := c.initializeChainInfo(ctx); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.HeadBlock(), headBlock) {
		t.Error("head block incorrect")
	}
	if !reflect.DeepEqual(c.HeadState(), headState) {
		t.Error("head block incorrect")
	}
	if headBlock.Slot != c.HeadSlot() {
		t.Error("head slot incorrect")
	}
	if !bytes.Equal(headRoot[:], c.HeadRoot()) {
		t.Error("head slot incorrect")
	}
}
