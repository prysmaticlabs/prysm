package blockchain

import (
	"context"
	"errors"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/casper"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	btestutil "github.com/prysmaticlabs/prysm/beacon-chain/testutil"
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

func setupBeaconChain(t *testing.T, faultyPoWClient bool, beaconDB *db.BeaconDB) *ChainService {
	endpoint := "ws://127.0.0.1"
	ctx := context.Background()
	var web3Service *powchain.Web3Service
	var err error
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
	if err := beaconDB.InitializeState(nil); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &Config{
		BeaconBlockBuf: 0,
		BeaconDB:       beaconDB,
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
	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db)

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
	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db)

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:        1,
		PowChainRef: []byte("a"),
	})

	blockChan := make(chan *types.Block)
	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing(blockChan)
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

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db)
	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState(nil)
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

	currentSlot := uint64(1)
	attestationSlot := uint64(0)
	shard := crystallized.ShardAndCommitteesForSlots()[attestationSlot].ArrayShardAndCommittee[0].Shard

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:                  currentSlot,
		ActiveStateRoot:       activeStateRoot[:],
		CrystallizedStateRoot: crystallizedStateRoot[:],
		AncestorHashes:        [][]byte{parentHash[:]},
		PowChainRef:           []byte("a"),
		Attestations: []*pb.AggregatedAttestation{{
			Slot:               attestationSlot,
			AttesterBitfield:   []byte{128, 0},
			Shard:              shard,
			JustifiedBlockHash: parentHash[:],
		}},
	})

	blockChan := make(chan *types.Block)
	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing(blockChan)
		<-exitRoutine
	}()

	if err := chainService.beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	chainService.incomingBlockChan <- block
	<-blockChan
	chainService.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Chain service context closed, exiting goroutine")
	testutil.AssertLogsContain(t, hook, "Processed block")
}

func TestDoesPOWBlockExist(t *testing.T) {
	hook := logTest.NewGlobal()
	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db)

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

func getShardForSlot(t *testing.T, cState *types.CrystallizedState, slot uint64) uint64 {
	shardAndCommittee, err := casper.GetShardAndCommitteesForSlot(
		cState.ShardAndCommitteesForSlots(),
		cState.LastStateRecalculationSlot(),
		slot)
	if err != nil {
		t.Fatalf("Unable to get shard for slot: %d", slot)
	}
	return shardAndCommittee.ArrayShardAndCommittee[0].Shard
}

func TestProcessBlocksWithCorrectAttestations(t *testing.T) {
	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db)
	active := types.NewGenesisActiveState()
	crystallized, err := types.NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	activeStateRoot, _ := active.Hash()
	crystallizedStateRoot, _ := crystallized.Hash()

	block0 := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	if saveErr := chainService.beaconDB.SaveBlock(block0); saveErr != nil {
		t.Fatalf("Could not save block: %v", saveErr)
	}
	block0Hash, err := block0.Hash()
	if err != nil {
		t.Fatalf("Failed to compute block's hash: %v", err)
	}

	currentSlot := uint64(1)
	attestationSlot := currentSlot - 1

	block1 := types.NewBlock(&pb.BeaconBlock{
		AncestorHashes:        [][]byte{block0Hash[:]},
		Slot:                  currentSlot,
		ActiveStateRoot:       activeStateRoot[:],
		CrystallizedStateRoot: crystallizedStateRoot[:],
		Attestations: []*pb.AggregatedAttestation{{
			Slot:               attestationSlot,
			AttesterBitfield:   []byte{128, 0},
			Shard:              getShardForSlot(t, crystallized, attestationSlot),
			JustifiedBlockHash: block0Hash[:],
		}},
	})

	blockChan := make(chan *types.Block)
	exitRoutine := make(chan bool)
	go func() {
		chainService.blockProcessing(blockChan)
		<-exitRoutine
	}()

	chainService.incomingBlockChan <- block1
	block1Returned := <-blockChan

	if block1 != block1Returned {
		t.Fatalf("expected %v and %v to be the same", block1, block1Returned)
	}

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
			{
				Slot:               currentSlot - 1,
				AttesterBitfield:   []byte{64, 0},
				Shard:              getShardForSlot(t, crystallized, currentSlot-1),
				JustifiedBlockHash: block0Hash[:],
			},
			{
				Slot:               currentSlot - 2,
				AttesterBitfield:   []byte{128, 0},
				Shard:              getShardForSlot(t, crystallized, currentSlot-2),
				JustifiedBlockHash: block0Hash[:],
			},
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
			{
				Slot:               currentSlot - 1,
				AttesterBitfield:   []byte{32, 0},
				Shard:              getShardForSlot(t, crystallized, currentSlot-1),
				JustifiedBlockHash: block0Hash[:],
			},
			{
				Slot:               currentSlot - 2,
				AttesterBitfield:   []byte{64, 0},
				Shard:              getShardForSlot(t, crystallized, currentSlot-2),
				JustifiedBlockHash: block0Hash[:],
			},
			{
				Slot:               currentSlot - 3,
				AttesterBitfield:   []byte{128, 0},
				Shard:              getShardForSlot(t, crystallized, currentSlot-3),
				JustifiedBlockHash: block0Hash[:],
			},
		}})

	chainService.incomingBlockChan <- block1
	<-blockChan
	chainService.incomingBlockChan <- block2
	<-blockChan
	chainService.incomingBlockChan <- block3
	<-blockChan

	chainService.cancel()
	exitRoutine <- true
}

func TestUpdateHead(t *testing.T) {
	genesisActive := types.NewGenesisActiveState()
	genesisCrystallized, err := types.NewGenesisCrystallizedState(nil)
	if err != nil {
		t.Fatalf("Could not generate genesis state: %v", err)
	}
	genesisActiveRoot, _ := genesisActive.Hash()
	genesisCrystallizedRoot, _ := genesisCrystallized.Hash()

	genesis := types.NewGenesisBlock(genesisActiveRoot, genesisCrystallizedRoot)
	genesisHash, err := genesis.Hash()
	if err != nil {
		t.Fatalf("Could not get genesis block hash: %v", err)
	}
	// Table driven tests for various fork choice scenarios.
	tests := []struct {
		blockSlot uint64
		aState    *types.ActiveState
		cState    *types.CrystallizedState
		logAssert string
	}{
		// Higher slot but same crystallized state should trigger chain update.
		{
			blockSlot: 64,
			aState:    genesisActive,
			cState:    genesisCrystallized,
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different crystallized state, but higher last finalized slot.
		{
			blockSlot: 64,
			aState:    genesisActive,
			cState:    types.NewCrystallizedState(&pb.CrystallizedState{LastFinalizedSlot: 10}),
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different crystallized state, same last finalized slot,
		// but last justified slot.
		{
			blockSlot: 64,
			aState:    genesisActive,
			cState: types.NewCrystallizedState(&pb.CrystallizedState{
				LastFinalizedSlot: 0,
				LastJustifiedSlot: 10,
			}),
			logAssert: "Chain head block and state updated",
		},
		// Same slot should not trigger a head update.
		{
			blockSlot: 0,
			aState:    genesisActive,
			cState:    genesisCrystallized,
			logAssert: "Chain head not updated",
		},
	}
	for _, tt := range tests {
		hook := logTest.NewGlobal()
		db := btestutil.SetupDB(t)
		defer btestutil.TeardownDB(t, db)
		chainService := setupBeaconChain(t, false, db)

		aRoot, _ := tt.aState.Hash()
		cRoot, _ := tt.cState.Hash()
		block := types.NewBlock(&pb.BeaconBlock{
			Slot:                  tt.blockSlot,
			ActiveStateRoot:       aRoot[:],
			CrystallizedStateRoot: cRoot[:],
			AncestorHashes:        [][]byte{genesisHash[:]},
			PowChainRef:           []byte("a"),
		})
		h, err := block.Hash()
		if err != nil {
			t.Fatal(err)
		}

		exitRoutine := make(chan bool)
		blockChan := make(chan *types.Block)
		go func() {
			chainService.updateHead(blockChan)
			<-exitRoutine
		}()

		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		chainService.unfinalizedBlocks[h] = &statePair{
			activeState:       tt.aState,
			crystallizedState: tt.cState,
			cycleTransition:   true,
		}

		// If blocks pending processing is empty, the updateHead routine does nothing.
		blockChan <- block
		chainService.cancel()
		exitRoutine <- true

		testutil.AssertLogsContain(t, hook, tt.logAssert)
	}
}
