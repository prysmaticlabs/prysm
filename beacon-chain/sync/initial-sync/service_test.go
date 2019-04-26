package initialsync

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockP2P struct {
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(ctx context.Context, msg proto.Message) {}

func (mp *mockP2P) Send(ctx context.Context, msg proto.Message, peerID peer.ID) error {
	return nil
}

type mockSyncService struct {
	hasStarted bool
	isSynced   bool
}

func (ms *mockSyncService) Start() {
	ms.hasStarted = true
}

func (ms *mockSyncService) IsSyncedWithNetwork() bool {
	return ms.isSynced
}

func (ms *mockSyncService) ResumeSync() {

}

type mockPowchain struct{}

func (mp *mockPowchain) BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error) {
	return true, nil, nil
}

type mockChainService struct{}

func (ms *mockChainService) CanonicalBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) ApplyBlockStateTransition(
	ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState,
) (*pb.BeaconState, error) {
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) VerifyBlockValidity(
	ctx context.Context,
	block *pb.BeaconBlock,
	beaconState *pb.BeaconState,
) error {
	return nil
}

func (ms *mockChainService) ApplyForkChoiceRule(ctx context.Context, block *pb.BeaconBlock, computedState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) CleanupBlockOperations(ctx context.Context, block *pb.BeaconBlock) error {
	return nil
}

func setUpGenesisStateAndBlock(beaconDB *db.BeaconDB, t *testing.T) {
	ctx := context.Background()
	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := beaconDB.InitializeState(context.Background(), unixTime, []*pb.Deposit{}, nil); err != nil {
		t.Fatalf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatalf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		log.Errorf("unable to marshal the beacon state: %v", err)
		return
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := beaconDB.SaveBlock(genBlock); err != nil {
		t.Fatalf("could not save genesis block to disk: %v", err)
	}
	if err := beaconDB.UpdateChainHead(ctx, genBlock, beaconState); err != nil {
		t.Fatalf("could not set chain head, %v", err)
	}
}

func TestSavingBlock_InSync(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		ChainService: &mockChainService{},
		BeaconDB:     db,
		PowChain:     &mockPowchain{},
	}
	ss := NewInitialSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	defer func() {
		close(exitRoutine)
	}()

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	genericHash := make([]byte, 32)
	genericHash[0] = 'a'

	fState := &pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisEpoch + 1,
		LatestBlock: &pb.BeaconBlock{
			Slot: params.BeaconConfig().GenesisSlot + params.BeaconConfig().SlotsPerEpoch,
		},
		LatestEth1Data: &pb.Eth1Data{
			BlockHash32: []byte{},
		},
	}

	stateResponse := &pb.BeaconStateResponse{
		FinalizedState: fState,
	}

	incorrectState := &pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisEpoch,
		JustifiedEpoch: params.BeaconConfig().GenesisEpoch + 1,
		LatestBlock: &pb.BeaconBlock{
			Slot: params.BeaconConfig().GenesisSlot + 4*params.BeaconConfig().SlotsPerEpoch,
		},
		LatestEth1Data: &pb.Eth1Data{
			BlockHash32: []byte{},
		},
	}

	incorrectStateResponse := &pb.BeaconStateResponse{
		FinalizedState: incorrectState,
	}

	stateRoot, err := hashutil.HashProto(fState)
	if err != nil {
		t.Fatalf("unable to tree hash state: %v", err)
	}
	beaconStateRootHash32 := stateRoot

	getBlockResponseMsg := func(Slot uint64) p2p.Message {
		block := &pb.BeaconBlock{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{1, 2, 3},
				BlockHash32:       []byte{4, 5, 6},
			},
			ParentRootHash32: genericHash,
			Slot:             Slot,
			StateRootHash32:  beaconStateRootHash32[:],
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		return p2p.Message{
			Peer: "",
			Data: blockResponse,
			Ctx:  context.Background(),
		}
	}

	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}

	msg1 := getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 1)

	// saving genesis block
	ss.blockBuf <- msg1

	msg2 := p2p.Message{
		Peer: "",
		Data: incorrectStateResponse,
		Ctx:  context.Background(),
	}

	ss.stateBuf <- msg2

	msg2.Data = stateResponse

	ss.stateBuf <- msg2

	msg1 = getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 1)
	ss.blockBuf <- msg1

	msg1 = getBlockResponseMsg(params.BeaconConfig().GenesisSlot + 2)
	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	hook.Reset()
	internal.TeardownDB(t, db)
}

func TestProcessingBatchedBlocks_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		ChainService: &mockChainService{},
		BeaconDB:     db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

	batchSize := 20
	batchedBlocks := make([]*pb.BeaconBlock, batchSize)
	expectedSlot := params.BeaconConfig().GenesisSlot + uint64(batchSize)

	for i := 1; i <= batchSize; i++ {
		batchedBlocks[i-1] = &pb.BeaconBlock{
			Slot: params.BeaconConfig().GenesisSlot + uint64(i),
		}
	}

	msg := p2p.Message{
		Ctx: context.Background(),
		Data: &pb.BatchedBeaconBlockResponse{
			BatchedBlocks: batchedBlocks,
		},
	}

	ss.processBatchedBlocks(msg)

	state, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := hashutil.HashProto(state)
	if err != nil {
		t.Fatal(err)
	}
	ss.highestObservedRoot = stateRoot

	if ss.currentSlot != expectedSlot {
		t.Errorf("Expected slot %d equal to current slot %d", expectedSlot, ss.currentSlot)
	}

	if ss.highestObservedSlot == expectedSlot {
		t.Errorf("Expected slot %d not equal to highest observed slot slot %d", expectedSlot, ss.highestObservedSlot)
	}
}

func TestProcessingBlocks_SkippedSlots(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	setUpGenesisStateAndBlock(db, t)
	ctx := context.Background()

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		ChainService: &mockChainService{},
		BeaconDB:     db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

	batchSize := 20
	expectedSlot := params.BeaconConfig().GenesisSlot + uint64(batchSize)
	ss.highestObservedSlot = expectedSlot
	blk, err := ss.db.BlockBySlot(ctx, params.BeaconConfig().GenesisSlot)
	if err != nil {
		t.Fatalf("Unable to get genesis block %v", err)
	}
	h, err := hashutil.HashBeaconBlock(blk)
	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}
	parentHash := h[:]
	state, err := db.HeadState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := hashutil.HashProto(state)
	if err != nil {
		t.Fatal(err)
	}
	ss.highestObservedRoot = stateRoot

	for i := 1; i <= batchSize; i++ {
		// skip slots
		if i == 4 || i == 6 || i == 13 || i == 17 {
			continue
		}
		block := &pb.BeaconBlock{
			Slot:             params.BeaconConfig().GenesisSlot + uint64(i),
			ParentRootHash32: parentHash,
		}

		ss.processBlock(context.Background(), block)

		// Save the block and set the parent hash of the next block
		// as the hash of the current block.
		if err := ss.db.SaveBlock(block); err != nil {
			t.Fatalf("Block unable to be saved %v", err)
		}

		hash, err := hashutil.HashBeaconBlock(block)
		if err != nil {
			t.Fatalf("Could not hash block %v", err)
		}
		parentHash = hash[:]
		state, err := db.HeadState(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		stateRoot, err := hashutil.HashProto(state)
		if err != nil {
			t.Fatal(err)
		}
		ss.highestObservedRoot = stateRoot
	}

	if ss.currentSlot != expectedSlot {
		t.Errorf("Expected slot %d equal to current slot %d", expectedSlot, ss.currentSlot)
	}

	if ss.highestObservedSlot != expectedSlot {
		t.Errorf("Expected slot %d equal to highest observed slot %d", expectedSlot, ss.highestObservedSlot)
	}
}

func TestSafelyHandleMessage(t *testing.T) {
	hook := logTest.NewGlobal()

	safelyHandleMessage(func(_ p2p.Message) {
		panic("bad!")
	}, p2p.Message{
		Data: &pb.BeaconBlock{},
	})

	testutil.AssertLogsContain(t, hook, "Panicked when handling p2p message!")
}

func TestSafelyHandleMessage_NoData(t *testing.T) {
	hook := logTest.NewGlobal()

	safelyHandleMessage(func(_ p2p.Message) {
		panic("bad!")
	}, p2p.Message{})

	entry := hook.LastEntry()
	if entry.Data["msg"] != "message contains no data" {
		t.Errorf("Message logged was not what was expected: %s", entry.Data["msg"])
	}
}
