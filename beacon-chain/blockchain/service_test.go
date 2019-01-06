package blockchain

import (
	"context"
	"errors"
	"io/ioutil"
	"math/big"
	"strconv"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/gogo/protobuf/proto"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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
		web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
			Endpoint: endpoint,
			Pubkey:   "",
			VrcAddr:  common.Address{},
			Reader:   client,
			Client:   client,
			Logger:   client,
		})
	} else {
		client := &mockClient{}
		web3Service, err = powchain.NewWeb3Service(ctx, &powchain.Web3ServiceConfig{
			Endpoint: endpoint,
			Pubkey:   "",
			VrcAddr:  common.Address{},
			Reader:   client,
			Client:   client,
			Logger:   client,
		})
	}
	if err != nil {
		t.Fatalf("unable to set up web3 service: %v", err)
	}
	if err := beaconDB.InitializeState(nil); err != nil {
		t.Fatalf("failed to initialize state: %v", err)
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

func SetSlotInState(service *ChainService, slot uint64) error {
	bState, err := service.beaconDB.GetState()
	if err != nil {
		return err
	}

	bState.Slot = slot
	return service.beaconDB.SaveState(bState)
}

func TestStartStop(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db)

	if err := SetSlotInState(chainService, 1); err != nil {
		t.Fatal(err)
	}

	parentBlock := &pb.BeaconBlock{
		Slot: 1,
	}

	parentHash, err := b.Hash(parentBlock)
	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}

	if err := chainService.beaconDB.SaveBlock(parentBlock); err != nil {
		t.Fatalf("Unable to save block %v", err)
	}

	block := &pb.BeaconBlock{
		Slot:                          2,
		ParentRootHash32:              parentHash[:],
		CandidatePowReceiptRootHash32: []byte("a"),
	}

	blockChan := make(chan *pb.BeaconBlock)
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

	testutil.AssertLogsContain(t, hook, "unable to retrieve POW chain reference block failed")
}

func TestRunningChainService(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db)
	deposits := make([]*pb.Deposit, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey: []byte(strconv.Itoa(i)),
			RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
				169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
		}
		depositData, err := b.EncodeDepositData(
			depositInput,
			params.BeaconConfig().MaxDepositInGwei,
			time.Now().Unix(),
		)
		if err != nil {
			t.Fatalf("Could not encode deposit input: %v", err)
		}
		deposits[i] = &pb.Deposit{DepositData: depositData}
	}
	beaconState, err := state.InitialBeaconState(deposits, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconState.Slot = 5

	enc, _ := proto.Marshal(beaconState)
	stateRoot := hashutil.Hash(enc)

	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("could not save block to db: %v", err)
	}
	parentHash, err := b.Hash(genesis)
	if err != nil {
		t.Fatalf("unable to get hash of canonical head: %v", err)
	}
	if err := chainService.beaconDB.SaveState(beaconState); err != nil {
		t.Fatalf("Can't save state to db %v", err)
	}
	beaconState, err = chainService.beaconDB.GetState()
	if err != nil {
		t.Fatalf("Can't get state from db %v", err)
	}

	var shardAndCommittees []*pb.ShardAndCommitteeArray
	for i := uint64(0); i < params.BeaconConfig().EpochLength*2; i++ {
		shardAndCommittees = append(shardAndCommittees, &pb.ShardAndCommitteeArray{
			ArrayShardAndCommittee: []*pb.ShardAndCommittee{
				{Committee: []uint32{9, 8, 311, 12, 92, 1, 23, 17}},
			},
		})
	}

	beaconState.ShardAndCommitteesAtSlots = shardAndCommittees
	if err := chainService.beaconDB.SaveState(beaconState); err != nil {
		t.Fatal(err)
	}

	currentSlot := uint64(5)
	attestationSlot := uint64(0)
	shard := beaconState.ShardAndCommitteesAtSlots[attestationSlot].ArrayShardAndCommittee[0].Shard

	block := &pb.BeaconBlock{
		Slot:                          currentSlot + 1,
		StateRootHash32:               stateRoot[:],
		ParentRootHash32:              parentHash[:],
		CandidatePowReceiptRootHash32: []byte("a"),
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{{
				ParticipationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &pb.AttestationData{
					Slot:                      attestationSlot,
					Shard:                     shard,
					JustifiedBlockRootHash32:  params.BeaconConfig().ZeroHash[:],
					LatestCrosslinkRootHash32: params.BeaconConfig().ZeroHash[:],
				},
			}},
		},
	}

	if err := SetSlotInState(chainService, currentSlot); err != nil {
		t.Fatal(err)
	}

	blockChan := make(chan *pb.BeaconBlock)
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
	testutil.AssertLogsContain(t, hook, "Processed beacon block")
}

func TestDoesPOWBlockExist(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, true, db)

	beaconState, err := chainService.beaconDB.GetState()
	if err != nil {
		t.Fatalf("Unable to retrieve beacon state %v", err)
	}

	// Using a faulty client should throw error.
	var powHash [32]byte
	copy(powHash[:], beaconState.ProcessedPowReceiptRootHash32)
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
	enc, _ := proto.Marshal(beaconState)
	stateRoot := hashutil.Hash(enc)

	genesis := b.NewGenesisBlock(stateRoot[:])
	genesisHash, err := b.Hash(genesis)
	if err != nil {
		t.Fatalf("Could not get genesis block hash: %v", err)
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
			state:     &pb.BeaconState{FinalizedSlot: 10},
			logAssert: "Chain head block and state updated",
		},
		// Higher slot, different crystallized state, same last finalized slot,
		// but last justified slot.
		{
			blockSlot: 64,
			state: &pb.BeaconState{
				FinalizedSlot: 0,
				JustifiedSlot: 10,
			},
			logAssert: "Chain head block and state updated",
		},
		// Same slot should not trigger a head update.
		{
			blockSlot: 0,
			state:     beaconState,
			logAssert: "Chain head not updated",
		},
	}
	for _, tt := range tests {
		hook := logTest.NewGlobal()
		db := internal.SetupDB(t)
		defer internal.TeardownDB(t, db)
		chainService := setupBeaconChain(t, false, db)

		enc, _ := proto.Marshal(tt.state)
		stateRoot := hashutil.Hash(enc)
		block := &pb.BeaconBlock{
			Slot:                          tt.blockSlot,
			StateRootHash32:               stateRoot[:],
			ParentRootHash32:              genesisHash[:],
			CandidatePowReceiptRootHash32: []byte("a"),
		}
		h, err := b.Hash(block)
		if err != nil {
			t.Fatal(err)
		}

		exitRoutine := make(chan bool)
		blockChan := make(chan *pb.BeaconBlock)
		go func() {
			chainService.updateHead(blockChan)
			<-exitRoutine
		}()

		if err := chainService.beaconDB.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		chainService.unfinalizedBlocks[h] = tt.state

		// If blocks pending processing is empty, the updateHead routine does nothing.
		blockChan <- block
		chainService.cancel()
		exitRoutine <- true

		testutil.AssertLogsContain(t, hook, tt.logAssert)
	}
}

func TestIsBlockReadyForProcessing(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	chainService := setupBeaconChain(t, false, db)
	beaconState, err := state.InitialBeaconState(nil, 0, nil)
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}

	block := &pb.BeaconBlock{
		ParentRootHash32: []byte{'a'},
	}

	if err := chainService.isBlockReadyForProcessing(block, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block having no parent saved")
	}

	beaconState.Slot = 10

	enc, _ := proto.Marshal(beaconState)
	stateRoot := hashutil.Hash(enc)
	genesis := b.NewGenesisBlock([]byte{})
	if err := chainService.beaconDB.SaveBlock(genesis); err != nil {
		t.Fatalf("cannot save block: %v", err)
	}
	parentHash, err := b.Hash(genesis)
	if err != nil {
		t.Fatalf("unable to get hash of canonical head: %v", err)
	}

	block2 := &pb.BeaconBlock{
		ParentRootHash32: parentHash[:],
		Slot:             10,
	}

	if err := chainService.isBlockReadyForProcessing(block2, beaconState); err == nil {
		t.Fatal("block processing succeeded despite block slot being invalid")
	}

	var h [32]byte
	copy(h[:], []byte("a"))
	beaconState.ProcessedPowReceiptRootHash32 = h[:]
	beaconState.Slot = 0

	currentSlot := uint64(1)
	attestationSlot := uint64(0)
	shard := beaconState.ShardAndCommitteesAtSlots[attestationSlot].ArrayShardAndCommittee[0].Shard

	block3 := &pb.BeaconBlock{
		Slot:                          currentSlot,
		StateRootHash32:               stateRoot[:],
		ParentRootHash32:              parentHash[:],
		CandidatePowReceiptRootHash32: []byte("a"),
		Body: &pb.BeaconBlockBody{
			Attestations: []*pb.Attestation{{
				ParticipationBitfield: []byte{128, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
					0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
				Data: &pb.AttestationData{
					Slot:                     attestationSlot,
					Shard:                    shard,
					JustifiedBlockRootHash32: parentHash[:],
				},
			}},
		},
	}

	chainService.enablePOWChain = true

	if err := chainService.isBlockReadyForProcessing(block3, beaconState); err != nil {
		t.Fatalf("block processing failed despite being a valid block: %v", err)
	}
}
