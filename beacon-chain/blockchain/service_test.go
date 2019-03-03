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

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/forkutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockOperationService struct{}

func (ms *mockOperationService) IncomingProcessedBlockFeed() *event.Feed {
	return new(event.Feed)
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
	domain := forkutils.DomainVersion(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
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

func setupBeaconChain(t *testing.T, faultyPoWClient bool, beaconDB *db.BeaconDB, enablePOWChain bool) *ChainService {
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
	chainService := setupBeaconChain(t, false, db, true)

	chainService.IncomingBlockFeed()

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
	chainService := setupBeaconChain(t, false, db, false)

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

	chainService := setupBeaconChain(t, false, db, true)

	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits); err != nil {
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

func TestChainService_FaultyPOWChain(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	if err := SetSlotInState(chainService, 1); err != nil {
		t.Fatal(err)
	}

	parentBlock := &pb.BeaconBlock{
		Slot: 1,
	}

	parentRoot, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("Unable to tree hash block %v", err)
	}

	if err := chainService.beaconDB.SaveBlock(parentBlock); err != nil {
		t.Fatalf("Unable to save block %v", err)
	}

	block := &pb.BeaconBlock{
		Slot:             2,
		ParentRootHash32: parentRoot[:],
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
	}

	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	chainService.incomingBlockChan <- block
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "unable to retrieve POW chain reference block")
}

func TestChainService_Starts(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)
	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := params.BeaconConfig().GenesisSlot
	beaconState.Slot++
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)

	block := &pb.BeaconBlock{
		Slot:             currentSlot + 1,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: nil,
		},
	}

	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	chainService.incomingBlockChan <- block
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Chain service context closed, exiting goroutine")
	testutil.AssertLogsContain(t, hook, "Processed beacon block")
}

func TestReceiveBlock_RemovesPendingDeposits(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	deposits, privKeys := setupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, genesisBlock := setupGenesisBlock(t, chainService, beaconState)
	beaconState.Slot++
	if err := chainService.beaconDB.UpdateChainHead(genesisBlock, beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := params.BeaconConfig().GenesisSlot
	randaoReveal := createRandaoReveal(t, beaconState, privKeys)

	pendingDeposits := []*pb.Deposit{
		createPreChainStartDeposit(t, []byte{'F'}),
	}
	depositTrie := trieutil.NewDepositTrie()
	for _, pd := range pendingDeposits {
		depositTrie.UpdateDepositTrie(pd.DepositData)
		pd.MerkleBranchHash32S = depositTrie.Branch()
	}
	depositRoot := depositTrie.Root()
	beaconState.LatestEth1Data.DepositRootHash32 = depositRoot[:]

	block := &pb.BeaconBlock{
		Slot:             currentSlot + 1,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Deposits: pendingDeposits,
		},
	}

	for _, dep := range pendingDeposits {
		db.InsertPendingDeposit(chainService.ctx, dep, big.NewInt(0))
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != len(pendingDeposits) || len(pendingDeposits) == 0 {
		t.Fatalf("Expected %d pending deposits", len(pendingDeposits))
	}

	beaconState.Slot--
	computedState, err := chainService.ReceiveBlock(block, beaconState)
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.ApplyForkChoiceRule(block, computedState); err != nil {
		t.Fatal(err)
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != 0 {
		t.Fatalf("Expected 0 pending deposits, but there are %+v", db.PendingDeposits(chainService.ctx, nil))
	}
	testutil.AssertLogsContain(t, hook, "Executing state transition")
}

func TestPOWBlockExists_UsingDepositRootHash(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, true, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(unixTime, deposits); err != nil {
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

func TestUpdateHead_SavesBlock(t *testing.T) {
	beaconState, err := state.GenesisBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("Cannot create genesis beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	genesisRoot, err := hashutil.HashProto(genesis)
	if err != nil {
		t.Fatalf("Could not get genesis block root: %v", err)
	}
	// Table driven tests for various fork choice scenarios.
	tests := []struct {
		blockSlot uint64
		state     *pb.BeaconState
		logAssert string
	}{
		// Higher slot but same crystallized state should trigger chain update.
		{
			blockSlot: 64,
			state:     beaconState,
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different crystallized state, but higher last finalized slot.
		{
			blockSlot: 64,
			state:     &pb.BeaconState{FinalizedEpoch: 2},
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different crystallized state, same last finalized slot,
		// but last justified slot.
		{
			blockSlot: 64,
			state: &pb.BeaconState{
				FinalizedEpoch: 0,
				JustifiedEpoch: 2,
			},
			logAssert: "Chain head block and state updated",
		},
	}
	for _, tt := range tests {
		hook := logTest.NewGlobal()
		db := internal.SetupDB(t)
		defer internal.TeardownDB(t, db)
		chainService := setupBeaconChain(t, false, db, true)
		unixTime := uint64(time.Now().Unix())
		deposits, _ := setupInitialDeposits(t, 100)
		if err := db.InitializeState(unixTime, deposits); err != nil {
			t.Fatalf("Could not initialize beacon state to disk: %v", err)
		}

		stateRoot, err := hashutil.HashProto(tt.state)
		if err != nil {
			t.Fatalf("Could not tree hash state: %v", err)
		}
		block := &pb.BeaconBlock{
			Slot:             tt.blockSlot,
			StateRootHash32:  stateRoot[:],
			ParentRootHash32: genesisRoot[:],
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte("a"),
				BlockHash32:       []byte("b"),
			},
		}
		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		if err := chainService.ApplyForkChoiceRule(block, tt.state); err != nil {
			t.Errorf("Expected head to update, received %v", err)
		}

		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		chainService.cancel()
		testutil.AssertLogsContain(t, hook, tt.logAssert)
	}
}

func TestIsBlockReadyForProcessing_ValidBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	chainService := setupBeaconChain(t, false, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits, privKeys := setupInitialDeposits(t, 100)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State(ctx)
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	block := &pb.BeaconBlock{
		ParentRootHash32: []byte{'a'},
	}

	if err := chainService.isBlockReadyForProcessing(block, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block having no parent saved")
	}

	beaconState.Slot = params.BeaconConfig().GenesisSlot + 10

	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("cannot save block: %v", err)
	}
	parentRoot, err := hashutil.HashBeaconBlock(genesis)
	if err != nil {
		t.Fatalf("unable to get root of canonical head: %v", err)
	}

	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: []byte{2},
		BlockHash32:       []byte{3},
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot

	currentSlot := params.BeaconConfig().GenesisSlot + 1
	attestationSlot := params.BeaconConfig().GenesisSlot

	randaoReveal := createRandaoReveal(t, beaconState, privKeys)
	block2 := &pb.BeaconBlock{
		Slot:             currentSlot,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentRoot[:],
		RandaoReveal:     randaoReveal,
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{{
				AggregationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &pb.AttestationData{
					Slot:                     attestationSlot,
					JustifiedBlockRootHash32: parentRoot[:],
				},
			}},
		},
	}

	chainService.enablePOWChain = true

	if err := chainService.isBlockReadyForProcessing(block2, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}
