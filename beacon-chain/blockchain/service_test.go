package blockchain

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
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

func setupBeaconChain(t *testing.T, faultyPoWClient bool) *ChainService {
	config := db.Config{Path: "", Name: "", InMemory: true}
	db, err := db.NewDB(config)
	if err != nil {
		t.Fatalf("could not setup beaconDB: %v", err)
	}

	endpoint := "ws://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Web3Service
	if faultyPoWClient {
		client := &faultyClient{}
		web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	} else {
		client := &mockClient{}
		web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{Endpoint: endpoint, Pubkey: "", VrcAddr: common.Address{}}, client, client, client)
	}
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       db,
		Web3Service:    web3Service,
		EnablePOWChain: true,
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

func TestStartStop(t *testing.T) {
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	chainService.IncomingBlockFeed()

	// Test the start function.
	chainService.Start()

	if err := chainService.Stop(); err != nil {
		t.Fatalf("unable to stop chain service: %v", err)
	}

	// The context should have been canceled.
	if chainService.ctx.Err() == nil {
		t.Error("context was not canceled")
	}
}

func TestRunningChainServiceFaultyPOWChain(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, true)
	defer chainService.beaconDB.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:        1,
		PowChainRef: []byte("a"),
	})

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

	testutil.AssertLogsContain(t, hook, "Proof-of-Work chain reference in block does not exist")
}

func TestRunningChainService(t *testing.T) {
	hook := logTest.NewGlobal()

	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	activeStateRoot, _ := active.Hash()
	crystallizedStateRoot, _ := crystallized.Hash()

	genesis := types.NewGenesisBlock([32]byte{}, [32]byte{})
	chainService.beaconDB.SaveBlock(genesis)
	parentHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("unable to get hash of canonical head: %v", err)
	}

	secondsSinceGenesis := time.Since(params.GetConfig().GenesisTime).Seconds()
	currentSlot := uint64(math.Floor(secondsSinceGenesis / float64(params.GetConfig().SlotDuration)))

	slotsStart := crystallized.LastStateRecalculationSlot() - params.GetConfig().CycleLength
	slotIndex := (currentSlot - slotsStart) % params.GetConfig().CycleLength
	Shard := crystallized.ShardAndCommitteesForSlots()[slotIndex].ArrayShardAndCommittee[0].Shard

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:                  currentSlot,
		ActiveStateRoot:       activeStateRoot[:],
		CrystallizedStateRoot: crystallizedStateRoot[:],
		AncestorHashes:        [][]byte{parentHash[:]},
		PowChainRef:           []byte("a"),
		Attestations: []*pb.AggregatedAttestation{{
			Slot:               currentSlot,
			AttesterBitfield:   []byte{128, 0},
			Shard:              Shard,
			JustifiedBlockHash: parentHash[:],
		}},
	})

	blockNoParent := types.NewBlock(&pb.BeaconBlock{
		Slot:           currentSlot,
		PowChainRef:    []byte("a"),
		AncestorHashes: [][]byte{{}},
	})

	exitRoutine := make(chan bool)
	t.Log([][]byte{parentHash[:]})
	go func() {
		chainService.blockProcessing()
		<-exitRoutine
	}()

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
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

func TestDoesPOWBlockExist(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, true)
	defer chainService.beaconDB.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot: 10,
	})

	// Using a faulty client should throw error.
	exists := chainService.doesPoWBlockExist(block)
	if exists {
		t.Error("Block corresponding to nil powchain reference should not exist")
	}
	testutil.AssertLogsContain(t, hook, "fetching PoW block corresponding to mainchain reference failed")
}

func TestUpdateHeadEmpty(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	ActiveStateRoot, _ := active.Hash()
	CrystallizedStateRoot, _ := crystallized.Hash()

	genesis := types.NewGenesisBlock(ActiveStateRoot, CrystallizedStateRoot)
	genesisHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("Could not get genesis block hash: %v", err)
	}

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:                  64,
		ActiveStateRoot:       ActiveStateRoot[:],
		CrystallizedStateRoot: CrystallizedStateRoot[:],
		AncestorHashes:        [][]byte{genesisHash[:]},
		PowChainRef:           []byte("a"),
	})

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	// If blocks pending processing is empty, the updateHead routine does nothing.
	chainService.blocksPendingProcessing = [][32]byte{}
	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsDoNotContain(t, hook, "Applying fork choice rule")
}

func TestUpdateHeadNoBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	// If blocks pending processing contains a hash of a block that does not exist
	// in persistent storage, we expect an error log to be thrown as
	// that is unexpected behavior given the block should have been saved during
	// processing.
	fakeBlock := types.NewBlock(&pb.BeaconBlock{Slot: 100})
	fakeBlockHash, err := fakeBlock.Hash()
	if err != nil {
		t.Fatal(err)
	}
	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, fakeBlockHash)
	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Could not get block")
}

func TestUpdateHeadNoParent(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	exitRoutine := make(chan bool)
	timeChan := make(chan time.Time)
	go func() {
		chainService.updateHead(timeChan)
		<-exitRoutine
	}()

	// non-existent parent hash should log an error in updateHead.
	noParentBlock := types.NewBlock(&pb.BeaconBlock{
		Slot:           64,
		AncestorHashes: [][]byte{{}},
	})
	noParentBlockHash, err := noParentBlock.Hash()
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}

	if err := chainService.beaconDB.SaveBlock(noParentBlock); err != nil {
		t.Fatal(err)
	}

	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, noParentBlockHash)

	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Failed to get parent of block %x", noParentBlockHash))
}

func TestUpdateHead(t *testing.T) {
	hook := logTest.NewGlobal()
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	ActiveStateRoot, _ := active.Hash()
	CrystallizedStateRoot, _ := crystallized.Hash()

	genesis := types.NewGenesisBlock(ActiveStateRoot, CrystallizedStateRoot)
	genesisHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("Could not get genesis block hash: %v", err)
	}

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:                  64,
		ActiveStateRoot:       ActiveStateRoot[:],
		CrystallizedStateRoot: CrystallizedStateRoot[:],
		AncestorHashes:        [][]byte{genesisHash[:]},
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

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	// If blocks pending processing is empty, the updateHead routine does nothing.
	chainService.blocksPendingProcessing = [][32]byte{}
	chainService.blocksPendingProcessing = append(chainService.blocksPendingProcessing, hash)
	timeChan <- time.Now()
	chainService.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Canonical block determined")
}

func TestProcessBlocksWithCorrectAttestations(t *testing.T) {
	chainService := setupBeaconChain(t, false)
	defer chainService.beaconDB.Close()

	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	ActiveStateRoot, _ := active.Hash()
	CrystallizedStateRoot, _ := crystallized.Hash()

	block0 := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	if saveErr := chainService.beaconDB.SaveBlock(block0); saveErr != nil {
		t.Fatalf("Cannot save block: %v", saveErr)
	}
	block0Hash, err := block0.Hash()
	if err != nil {
		t.Fatalf("Failed to compute block's hash: %v", err)
	}

	secondsSinceGenesis := time.Since(params.GetConfig().GenesisTime).Seconds()
	currentSlot := uint64(math.Floor(secondsSinceGenesis / float64(params.GetConfig().SlotDuration)))

	block1 := types.NewBlock(&pb.BeaconBlock{
		AncestorHashes:        [][]byte{block0Hash[:]},
		Slot:                  currentSlot,
		ActiveStateRoot:       ActiveStateRoot[:],
		CrystallizedStateRoot: CrystallizedStateRoot[:],
		Attestations: []*pb.AggregatedAttestation{{
			Slot:             currentSlot,
			AttesterBitfield: []byte{16, 0},
			Shard:            0,
		}},
	})

	exitRoutine := make(chan bool)
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
		AncestorHashes: [][]byte{block1Hash[:]},
		Slot:           currentSlot,
		Attestations: []*pb.AggregatedAttestation{
			{Slot: currentSlot - 1, AttesterBitfield: []byte{8, 0}, Shard: 0},
			{Slot: currentSlot, AttesterBitfield: []byte{8, 0}, Shard: 0},
		}})
	block2Hash, err := block2.Hash()
	if err != nil {
		t.Fatalf("unable to get hash of block 1: %v", err)
	}

	currentSlot++

	// Add 1 more attestation field for slot3
	block3 := types.NewBlock(&pb.BeaconBlock{
		AncestorHashes: [][]byte{block2Hash[:]},
		Slot:           currentSlot,
		Attestations: []*pb.AggregatedAttestation{
			{Slot: currentSlot - 2, AttesterBitfield: []byte{4, 0}, Shard: 0},
			{Slot: currentSlot - 1, AttesterBitfield: []byte{4, 0}, Shard: 0},
			{Slot: currentSlot, AttesterBitfield: []byte{4, 0}, Shard: 0},
		}})

	chainService.incomingBlockChan <- block1
	chainService.incomingBlockChan <- block2
	chainService.incomingBlockChan <- block3

	chainService.cancel()
	exitRoutine <- true
}
