package blockchain

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/attestation"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/forkutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure ChainService implements interfaces.
var _ = ChainFeeds(&ChainService{})

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
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

func (mb *mockBroadcaster) Broadcast(_ context.Context, _ proto.Message) {
	mb.broadcastCalled = true
}

var _ = p2p.Broadcaster(&mockBroadcaster{})

func setupInitialDeposits(t *testing.T, numDeposits int) ([]*pb.Deposit, []*bls.SecretKey) {
	privKeys := make([]*bls.SecretKey, numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		priv, err := bls.RandKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		depositInput := &pb.DepositInput{
			Pubkey: priv.PublicKey().Marshal(),
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			t.Fatalf("Cannot encode data: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
		privKeys[i] = priv
	}
	return deposits, privKeys
}

func createPreChainStartDeposit(t *testing.T, pk []byte) *pb.Deposit {
	depositInput := &pb.DepositInput{Pubkey: pk}
	balance := params.BeaconConfig().MaxDepositAmount
	depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
	if err != nil {
		t.Fatalf("Cannot encode data: %v", err)
	}
	return &pb.Deposit{DepositData: depositData}
}

func createRandaoReveal(t *testing.T, beaconState *pb.BeaconState, privKeys []*bls.SecretKey) []byte {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState, beaconState.Slot)
	if err != nil {
		t.Fatal(err)
	}
	epoch := helpers.SlotToEpoch(beaconState.Slot)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := forkutil.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal()
}

func setupGenesisBlock(t *testing.T, cs *ChainService, beaconState *pb.BeaconState) ([32]byte, *pb.BeaconBlock) {
	genesis := b.NewGenesisBlock([]byte{})
	if err := cs.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("could not save block to db: %v", err)
	}
	parentHash, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		t.Fatalf("unable to get tree hash root of canonical head: %v", err)
	}
	return parentHash, genesis
}

func setupBeaconChain(t *testing.T, faultyPoWClient bool, beaconDB *db.BeaconDB, enablePOWChain bool, attsService *attestation.Service) *ChainService {
	endpoint := "ws://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Web3Service
	var err error
	if enablePOWChain {
		if faultyPoWClient {
			client := &faultyClient{}
			web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
				Endpoint:        endpoint,
				DepositContract: common.Address{},
				Reader:          client,
				Client:          client,
				Logger:          client,
			})
		} else {
			client := &mockClient{}
			web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
				Endpoint:        endpoint,
				DepositContract: common.Address{},
				Reader:          client,
				Client:          client,
				Logger:          client,
			})
		}
	}
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       beaconDB,
		Web3Service:    web3Service,
		OpsPoolService: &mockOperationService{},
		EnablePOWChain: enablePOWChain,
		AttsService:    attsService,
		P2p:            &mockBroadcaster{},
	}
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, err := NewChainService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	return chainService
}

func SetSlotInState(service *ChainService, slot uint64) error {
	bState, err := service.beaconDB.State(context.Background())
	if err != nil {
		return err
	}

	bState.Slot = slot
	return service.beaconDB.SaveState(bState)
}

func TestChainStartStop_Uninitialized(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true, nil)

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

	beaconState, err := db.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if beaconState == nil || beaconState.Slot != params.BeaconConfig().GenesisSlot {
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

func TestChainStartStop_UninitializedAndNoPOWChain(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, false, nil)

	origExitFunc := logrus.StandardLogger().ExitFunc
	defer func() { logrus.StandardLogger().ExitFunc = origExitFunc }()
	fatal := false
	logrus.StandardLogger().ExitFunc = func(int) { fatal = true }
	// Test the start function.
	chainService.Start()

	if !fatal {
		t.Fatalf("Not exists fatal for init BeaconChain without POW chain")
	}
	testutil.AssertLogsContain(t, hook, "Not configured web3Service for POW chain")
}

func TestChainStartStop_Initialized(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, false, db, true, nil)

	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits, &pb.Eth1Data{}); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State(ctx)
	if err != nil {
		t.Fatalf("Could not fetch beacon state: %v", err)
	}
	setupGenesisBlock(t, chainService, beaconState)
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

func TestPOWBlockExists_UsingDepositRootHash(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, true, db, true, nil)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	eth1Data := &pb.Eth1Data{
		DepositRootHash32: []byte{},
		BlockHash32:       []byte{},
	}
	if err := db.InitializeState(unixTime, deposits, eth1Data); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	beaconState, err := chainService.beaconDB.State(ctx)
	if err != nil {
		t.Fatalf("Unable to retrieve beacon state %v", err)
	}

	// Using a faulty client should throw error.
	powHash := bytesutil.ToBytes32(beaconState.LatestEth1Data.DepositRootHash32)
	exists := chainService.doesPoWBlockExist(powHash)
	if exists {
		t.Error("Block corresponding to nil powchain reference should not exist")
	}
	testutil.AssertLogsContain(t, hook, "fetching PoW block corresponding to mainchain reference failed")
}
