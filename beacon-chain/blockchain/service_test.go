package blockchain

import (
	"context"
	"errors"
	"io/ioutil"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockClient struct{}

func (f *mockClient) SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *mockClient) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	head := &gethTypes.Header{Number: big.NewInt(0), Difficulty: big.NewInt(100)}
	return gethTypes.NewBlockWithHeader(head), nil
}

func (f *mockClient) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	return new(event.Feed).Subscribe(ch), nil
}

func (f *mockClient) LatestBlockHash() common.Hash {
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

func (f *faultyClient) LatestBlockHash() common.Hash {
	return common.BytesToHash([]byte{'A'})
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestStartStop(t *testing.T) {
	ctx := context.Background()

	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
	}
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	chainService, err := NewChainService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.slotAlignmentDuration = 0

	chainService.IncomingBlockFeed()
	chainService.CanonicalBlockBySlotNumber(0)
	chainService.CheckForCanonicalBlockBySlot(0)
	chainService.CanonicalHead()
	chainService.CanonicalCrystallizedState()

	// Test the start function.
	chainService.Start()

	cfg = &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}
	chainService, err = NewChainService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.slotAlignmentDuration = 0

	chainService.Start()

	if len(chainService.CurrentActiveState().RecentBlockHashes()) != 128 {
		t.Errorf("incorrect recent block hashes")
	}

	if len(chainService.CurrentCrystallizedState().Validators()) != params.GetConfig().BootstrappedValidatorsCount {
		t.Errorf("incorrect default validator size")
	}
	blockExists, err := chainService.ContainsBlock([32]byte{})
	if err != nil {
		t.Fatalf("unable to check if block exists: %v", err)
	}
	if blockExists {
		t.Errorf("chain is not empty")
	}
	hasState, err := chainService.HasStoredState()
	if err != nil {
		t.Fatalf("calling HasStoredState failed")
	}
	if hasState {
		t.Errorf("has stored state should return false")
	}
	chainService.CanonicalBlockFeed()
	chainService.CanonicalCrystallizedStateFeed()

	chainService, _ = NewChainService(ctx, cfg)

	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}}, make(map[[32]byte]*types.VoteCache))

	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}
	chainService.chain.SetActiveState(active)

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}
	chainService.chain.SetCrystallizedState(crystallized)

	parentBlock := types.NewBlock(nil)
	parentHash, _ := parentBlock.Hash()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
		PowChainRef:           []byte("a"),
	})
	if err := chainService.SaveBlock(block); err != nil {
		t.Errorf("save block should have failed")
	}

	// Save states so HasStoredState state should return true.
	chainService.chain.SetActiveState(types.NewActiveState(&pb.ActiveState{}, make(map[[32]byte]*types.VoteCache)))
	chainService.chain.SetCrystallizedState(types.NewCrystallizedState(&pb.CrystallizedState{}))
	hasState, _ = chainService.HasStoredState()
	if !hasState {
		t.Errorf("has stored state should return false")
	}

	if err := chainService.Stop(); err != nil {
		t.Fatalf("unable to stop chain service: %v", err)
	}

	// The context should have been canceled.
	if chainService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
}

func TestFaultyStop(t *testing.T) {
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}

	chainService, err := NewChainService(ctx, cfg)
	if err != nil {
		t.Fatalf("unable to setup chain service: %v", err)
	}

	chainService.slotAlignmentDuration = 0

	chainService.Start()

	chainService.chain.SetActiveState(types.NewActiveState(nil, make(map[[32]byte]*types.VoteCache)))

	err = chainService.Stop()
	if err == nil {
		t.Errorf("chain stop should have failed with persist active state")
	}

	chainService.chain.SetActiveState(types.NewActiveState(&pb.ActiveState{}, make(map[[32]byte]*types.VoteCache)))

	chainService.chain.SetCrystallizedState(types.NewCrystallizedState(nil))
	err = chainService.Stop()
	if err == nil {
		t.Errorf("chain stop should have failed with persist crystallized state")
	}
}

func TestCurrentBeaconSlot(t *testing.T) {
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &faultyClient{}
	web3Service, err := powchain.NewWeb3Service(
		ctx,
		&powchain.Web3ServiceConfig{
			Endpoint: endpoint,
			Pubkey:   "",
			VrcAddr:  common.Address{},
		},
		client,
		client,
		client,
	)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)
	chainService.genesisTimestamp = time.Now()
	if chainService.CurrentBeaconSlot() != 0 {
		t.Errorf("Expected us to be in the 0th slot, received %v", chainService.CurrentBeaconSlot())
	}
}

func TestRunningChainServiceFaultyPOWChain(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &faultyClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  1,
		PowChainRef: []byte("a"),
	})

	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	if err := chainService.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	chainService.incomingBlockChan <- block
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Proof-of-Work chain reference in block does not exist")
}

func TestRunningChainService(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	activeStateHash, _ := active.Hash()
	crystallizedStateHash, _ := crystallized.Hash()

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)

	genesis, err := beaconChain.GenesisBlock()
	if err != nil {
		t.Fatalf("unable to get canonical head: %v", err)
	}
	beaconChain.saveBlock(genesis)
	parentHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("unable to get hash of canonical head: %v", err)
	}

	secondsSinceGenesis := time.Since(params.GetConfig().GenesisTime).Seconds()
	currentSlot := uint64(math.Floor(secondsSinceGenesis / float64(params.GetConfig().SlotDuration)))

	slotsStart := crystallized.LastStateRecalc() - params.GetConfig().CycleLength
	slotIndex := (currentSlot - slotsStart) % params.GetConfig().CycleLength
	shardID := crystallized.ShardAndCommitteesForSlots()[slotIndex].ArrayShardAndCommittee[0].ShardId

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:            currentSlot,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
		PowChainRef:           []byte("a"),
		Attestations: []*pb.AggregatedAttestation{{
			Slot:               currentSlot,
			AttesterBitfield:   []byte{128, 0},
			ShardId:            shardID,
			JustifiedBlockHash: parentHash[:],
		}},
	})

	blockNoParent := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  currentSlot,
		PowChainRef: []byte("a"),
	})

	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	if err := chainService.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	chainService.incomingBlockChan <- blockNoParent
	chainService.incomingBlockChan <- block
	chainService.cancel()
	exitRoutine <- true
	testutil.WaitForLog(t, hook, "Chain service context closed, exiting goroutine")
	testutil.AssertLogsContain(t, hook, "Block points to nil parent")
	testutil.AssertLogsContain(t, hook, "Finished processing received block")
}

func TestBlockSlotNumberByHash(t *testing.T) {
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 1,
	})
	hash, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if err := chainService.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	slot, err := chainService.BlockSlotNumberByHash(hash)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if slot != 1 {
		t.Errorf("Expected slot 1, received %d", slot)
	}
	_, err = chainService.BlockSlotNumberByHash([32]byte{})
	if !strings.Contains(err.Error(), "could not get block from DB") {
		t.Errorf("Received incorrect error, expected could not get block from DB, received %v", err)
	}
}

func TestDoesPOWBlockExist(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &faultyClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf:   0,
		IncomingBlockBuf: 0,
		BeaconDB:         db.DB(),
		Chain:            beaconChain,
		Web3Service:      web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)
	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 10,
	})

	// Using a faulty client should throw error.
	exists := chainService.doesPoWBlockExist(block)
	if exists {
		t.Error("Block corresponding to nil powchain reference should not exist")
	}
	testutil.AssertLogsContain(t, hook, "fetching PoW block corresponding to mainchain reference failed")
}

func TestUpdateHead(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)

	}
	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf:   0,
		IncomingBlockBuf: 0,
		BeaconDB:         db.DB(),
		Chain:            beaconChain,
		Web3Service:      web3Service,
	}
	chainService, _ := NewChainService(ctx, cfg)

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	activeStateHash, _ := active.Hash()
	crystallizedStateHash, _ := crystallized.Hash()

	genesis := types.NewGenesisBlock(activeStateHash, crystallizedStateHash)
	genesisHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("Could not get genesis block hash: %v", err)
	}

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:            64,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            genesisHash[:],
		PowChainRef:           []byte("a"),
	})
	hash, err := block.Hash()
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	if err := chainService.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	// If blocks pending processing is empty, the updateHead routine does nothing.
	chainService.blocksPendingProcessing = [][32]byte{}
	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	chainService, _ = NewChainService(ctx, cfg)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	// If blocks pending processing contains a hash of a block that does not exist
	// in persistent storage, we expect an error log to be thrown as
	// that is unexpected behavior given the block should have been saved during
	// processing.
	fakeBlock := types.NewBlock(&pb.BeaconBlock{SlotNumber: 100})
	fakeBlockHash, err := fakeBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}
	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, fakeBlockHash)
	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	// Inexistent parent hash should log an error in updateHead.
	noParentBlock := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 64,
	})
	noParentBlockHash, err := noParentBlock.Hash()
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	chainService, _ = NewChainService(ctx, cfg)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	if err := chainService.SaveBlock(noParentBlock); err != nil {
		t.Fatal(err)
	}

	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, noParentBlockHash)

	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	// Now we test the correct, end-to-end updateHead functionality.
	chainService, _ = NewChainService(ctx, cfg)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, hash)

	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Could not get block")
	testutil.AssertLogsContain(t, hook, "Failed to get parent of block")
	testutil.AssertLogsContain(t, hook, "Canonical block determined")
}

func TestProcessBlocksWithCorrectAttestations(t *testing.T) {
	ctx := context.Background()
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	endpoint := "ws://127.0.0.1"
	client := &mockClient{}
	web3Service, err := powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	beaconChain, err := NewBeaconChain("", db.DB())
	if err != nil {
		t.Fatalf("could not register blockchain service: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db.DB(),
		Chain:          beaconChain,
		Web3Service:    web3Service,
	}

	chainService, _ := NewChainService(ctx, cfg)

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	activeStateHash, _ := active.Hash()
	crystallizedStateHash, _ := crystallized.Hash()

	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	block0 := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 0,
	})
	if saveErr := beaconChain.saveBlock(block0); saveErr != nil {
		t.Fatalf("Cannot save block: %v", saveErr)
	}
	block0Hash, err := block0.Hash()
	if err != nil {
		t.Fatalf("Failed to compute block's hash: %v", err)
	}

	secondsSinceGenesis := time.Since(params.GetConfig().GenesisTime).Seconds()
	currentSlot := uint64(math.Floor(secondsSinceGenesis / float64(params.GetConfig().SlotDuration)))

	block1 := types.NewBlock(&pb.BeaconBlock{
		ParentHash:            block0Hash[:],
		SlotNumber:            currentSlot,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		Attestations: []*pb.AggregatedAttestation{{
			Slot:             currentSlot,
			AttesterBitfield: []byte{16, 0},
			ShardId:          0,
		}},
	})

	exitRoutine = make(chan bool)
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	chainService.incomingBlockChan <- block1

	block1Hash, err := block1.Hash()
	if err != nil {
		t.Fatalf("unable to get hash of block 1: %v", err)
	}

	currentSlot++

	// Add 1 more attestation field for slot2
	block2 := types.NewBlock(&pb.BeaconBlock{
		ParentHash: block1Hash[:],
		SlotNumber: currentSlot,
		Attestations: []*pb.AggregatedAttestation{
			{Slot: currentSlot - 1, AttesterBitfield: []byte{8, 0}, ShardId: 0},
			{Slot: currentSlot, AttesterBitfield: []byte{8, 0}, ShardId: 0},
		}})
	block2Hash, err := block2.Hash()
	if err != nil {
		t.Fatalf("unable to get hash of block 1: %v", err)
	}

	currentSlot++

	// Add 1 more attestation field for slot3
	block3 := types.NewBlock(&pb.BeaconBlock{
		ParentHash: block2Hash[:],
		SlotNumber: currentSlot,
		Attestations: []*pb.AggregatedAttestation{
			{Slot: currentSlot - 2, AttesterBitfield: []byte{4, 0}, ShardId: 0},
			{Slot: currentSlot - 1, AttesterBitfield: []byte{4, 0}, ShardId: 0},
			{Slot: currentSlot, AttesterBitfield: []byte{4, 0}, ShardId: 0},
		}})

	chainService.incomingBlockChan <- block1
	chainService.incomingBlockChan <- block2
	chainService.incomingBlockChan <- block3

	chainService.cancel()
	exitRoutine <- true
}
