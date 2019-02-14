package blockchain

import (
	"context"
	"errors"
	"io/ioutil"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/shared/ssz"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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

type mockClient struct{}

func (m *mockClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (m *mockClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	head := &gethTypes.Header{Number: big.NewInt(0), Difficulty: big.NewInt(100)}
	return gethTypes.NewBlockWithHeader(head), nil
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

func setupInitialDeposits(t *testing.T) []*pb.Deposit {
	genesisValidatorRegistry := validators.InitialValidatorRegistry()
	deposits := make([]*pb.Deposit, len(genesisValidatorRegistry))
	for i := 0; i < len(deposits); i++ {
		deposits[i] = createPreChainStartDeposit(t, genesisValidatorRegistry[i].Pubkey)
	}
	return deposits
}

func createPreChainStartDeposit(t *testing.T, pk []byte) *pb.Deposit {
	depositInput := &pb.DepositInput{Pubkey: pk}
	balance := params.BeaconConfig().MaxDepositAmount
	depositData, err := b.EncodeDepositData(depositInput, balance, time.Now().Unix())
	if err != nil {
		t.Fatalf("Cannot encode data: %v", err)
	}
	return &pb.Deposit{DepositData: depositData}
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
	bState, err := service.beaconDB.State()
	if err != nil {
		return err
	}

	bState.Slot = slot
	return service.beaconDB.SaveState(bState)
}

func TestStartStopUninitializedChain(t *testing.T) {
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

func TestStartUninitializedChainWithoutConfigPOWChain(t *testing.T) {
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

func TestStartStopInitializedChain(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)

	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State()
	if err != nil {
		t.Fatalf("Could not fetch beacon state: %v", err)
	}
	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash beacon state: %v", err)
	}
	if err := db.SaveBlock(b.NewGenesisBlock(stateRoot[:])); err != nil {
		t.Fatalf("Could not save genesis block to disk: %v", err)
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

func TestRunningChainServiceFaultyPOWChain(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	if err := SetSlotInState(chainService, 1); err != nil {
		t.Fatal(err)
	}

	parentBlock := &pb.BeaconBlock{
		Slot: 1,
	}

	parentRoot, err := ssz.TreeHash(parentBlock)
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

	testutil.AssertLogsContain(t, hook, "unable to retrieve POW chain reference block failed")
}

func setupGenesisState(t *testing.T, cs *ChainService, beaconState *pb.BeaconState) ([32]byte, *pb.BeaconState) {
	genesis := b.NewGenesisBlock([]byte{})
	if err := cs.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("could not save block to db: %v", err)
	}
	parentHash, err := ssz.TreeHash(genesis)
	if err != nil {
		t.Fatalf("unable to get tree hash root of canonical head: %v", err)
	}
	if err := cs.beaconDB.SaveState(beaconState); err != nil {
		t.Fatalf("Can't save state to db %v", err)
	}
	beaconState, err = cs.beaconDB.State()
	if err != nil {
		t.Fatalf("Can't get state from db %v", err)
	}

	return parentHash, beaconState
}
func TestRunningChainService(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.Slot = 5

	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, beaconState := setupGenesisState(t, chainService, beaconState)

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			Pubkey:    []byte(strconv.Itoa(i)),
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	beaconState.ValidatorRegistry = validators
	if err := chainService.beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := params.BeaconConfig().GenesisSlot + 5
	attestationSlot := params.BeaconConfig().GenesisSlot

	block := &pb.BeaconBlock{
		Slot:             currentSlot + 1,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentHash[:],
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
					JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
					JustifiedEpoch:           currentSlot / params.BeaconConfig().EpochLength,
					LatestCrosslink: &pb.Crosslink{
						Epoch:                currentSlot / params.BeaconConfig().EpochLength,
						ShardBlockRootHash32: params.BeaconConfig().ZeroHash[:]},
				},
			}},
		},
	}

	if err := SetSlotInState(chainService, currentSlot); err != nil {
		t.Fatal(err)
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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 5

	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	parentHash, beaconState := setupGenesisState(t, chainService, beaconState)

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	beaconState.ValidatorRegistry = validators
	if err := chainService.beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}
	currentSlot := params.BeaconConfig().GenesisSlot + 5
	attestationSlot := params.BeaconConfig().GenesisSlot

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
		Eth1Data: &pb.Eth1Data{
			DepositRootHash32: []byte("a"),
			BlockHash32:       []byte("b"),
		},
		Body: &pb.BeaconBlockBody{
			Deposits: pendingDeposits,
			Attestations: []*pb.Attestation{{
				AggregationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &pb.AttestationData{
					Slot:                     attestationSlot,
					JustifiedBlockRootHash32: params.BeaconConfig().ZeroHash[:],
					JustifiedEpoch:           currentSlot / params.BeaconConfig().EpochLength,
					LatestCrosslink: &pb.Crosslink{
						Epoch:                currentSlot / params.BeaconConfig().EpochLength,
						ShardBlockRootHash32: params.BeaconConfig().ZeroHash[:]},
				},
			}},
		},
	}

	if err := SetSlotInState(chainService, currentSlot); err != nil {
		t.Fatal(err)
	}

	for _, dep := range pendingDeposits {
		db.InsertPendingDeposit(chainService.ctx, dep, big.NewInt(0))
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != len(pendingDeposits) || len(pendingDeposits) == 0 {
		t.Fatalf("Expected %d pending deposits", len(pendingDeposits))
	}

	_, err = chainService.ReceiveBlock(block, beaconState)
	if err != nil {
		t.Fatal(err)
	}

	if len(db.PendingDeposits(chainService.ctx, nil)) != 0 {
		t.Fatalf("Expected 0 pending deposits, but there are %+v", db.PendingDeposits(chainService.ctx, nil))
	}
}

func TestDoesPOWBlockExist(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}

	beaconState, err := chainService.beaconDB.State()
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

func TestUpdateHead(t *testing.T) {
	beaconState, err := state.InitialBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("Cannot create genesis beacon state: %v", err)
	}
	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}

	genesis := b.NewGenesisBlock(stateRoot[:])
	genesisRoot, err := ssz.TreeHash(genesis)
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
		deposits := setupInitialDeposits(t)
		if err := db.InitializeState(unixTime, deposits); err != nil {
			t.Fatalf("Could not initialize beacon state to disk: %v", err)
		}

		stateRoot, err := ssz.TreeHash(tt.state)
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

func TestIsBlockReadyForProcessing(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db, true)
	unixTime := uint64(time.Now().Unix())
	deposits := setupInitialDeposits(t)
	if err := db.InitializeState(unixTime, deposits); err != nil {
		t.Fatalf("Could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State()
	if err != nil {
		t.Fatalf("Can't get genesis state: %v", err)
	}
	block := &pb.BeaconBlock{
		ParentRootHash32: []byte{'a'},
	}

	if err := chainService.isBlockReadyForProcessing(block, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block having no parent saved")
	}

	beaconState.Slot = 10

	stateRoot, err := ssz.TreeHash(beaconState)
	if err != nil {
		t.Fatalf("Could not tree hash state: %v", err)
	}
	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("cannot save block: %v", err)
	}
	parentRoot, err := ssz.TreeHash(genesis)
	if err != nil {
		t.Fatalf("unable to get root of canonical head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		ParentRootHash32: parentRoot[:],
		Slot:             10,
	}

	if err := chainService.isBlockReadyForProcessing(block2, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block slot being invalid")
	}

	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: []byte{2},
		BlockHash32:       []byte{3},
	}
	beaconState.Slot = 0

	currentSlot := uint64(1)
	attestationSlot := uint64(0)

	block3 := &pb.BeaconBlock{
		Slot:             currentSlot,
		StateRootHash32:  stateRoot[:],
		ParentRootHash32: parentRoot[:],
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

	if err := chainService.isBlockReadyForProcessing(block3, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}
