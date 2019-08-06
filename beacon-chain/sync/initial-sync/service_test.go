package initialsync

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

func (mp *mockP2P) Reputation(_ peer.ID, _ int) {

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

type mockChainService struct {
	headBlock *ethpb.BeaconBlock
	headState *pb.BeaconState
}

func (ms *mockChainService) CanonicalRoot(slot uint64) []byte {
	return nil
}

func (ms *mockChainService) FinalizedState(ctx context.Context) (*pb.BeaconState, error) {
	return nil, nil
}

func (ms *mockChainService) FinalizedBlock() (*ethpb.BeaconBlock, error) {
	return nil, nil

}

func (ms *mockChainService) FinalizedCheckpt() *ethpb.Checkpoint {
	return &ethpb.Checkpoint{}
}

func (ms *mockChainService) JustifiedCheckpt() *ethpb.Checkpoint {
	return nil
}

func (ms *mockChainService) HeadSlot() uint64 {
	return 0
}

func (ms *mockChainService) HeadRoot() []byte {
	return nil
}

func (ms *mockChainService) HeadBlock() (*ethpb.BeaconBlock, error) {
	return ms.headBlock, nil
}

func (ms *mockChainService) HeadState() (*pb.BeaconState, error) {
	return ms.headState, nil
}

func (ms *mockChainService) CanonicalBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	return nil
}

func (ms *mockChainService) AdvanceState(
	ctx context.Context, beaconState *pb.BeaconState, block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	return &pb.BeaconState{
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}, nil
}

func (ms *mockChainService) VerifyBlockValidity(
	ctx context.Context,
	block *ethpb.BeaconBlock,
	beaconState *pb.BeaconState,
) error {
	return nil
}

func (ms *mockChainService) ApplyForkChoiceRule(ctx context.Context, block *ethpb.BeaconBlock, computedState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) CleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func setUpGenesisStateAndBlock(beaconDB *db.BeaconDB, t *testing.T) (*ethpb.BeaconBlock, *pb.BeaconState) {
	ctx := context.Background()
	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := beaconDB.InitializeState(context.Background(), unixTime, []*ethpb.Deposit{}, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatalf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Errorf("unable to marshal the beacon state: %v", err)
		return nil, nil
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := beaconDB.SaveBlock(genBlock); err != nil {
		t.Fatalf("could not save genesis block to disk: %v", err)
	}
	return genBlock, beaconState
}

func TestProcessingBatchedBlocks_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	b, s := setUpGenesisStateAndBlock(db, t)

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		ChainService: &mockChainService{headState: s, headBlock: b},
		BeaconDB:     db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

	batchSize := 20
	batchedBlocks := make([]*ethpb.BeaconBlock, batchSize)
	blocks, err := ss.db.BlocksBySlot(ss.ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	genBlock := blocks[0]
	parentRoot, err := ssz.SigningRoot(genBlock)
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= batchSize; i++ {
		block := &ethpb.BeaconBlock{
			Slot:       uint64(i),
			ParentRoot: parentRoot[:],
		}
		batchedBlocks[i-1] = block
		parentRoot, err = ssz.SigningRoot(block)
		if err != nil {
			t.Fatal(err)
		}
	}
	// edge case: handle out of order block list. Specifically with the highest
	// block first. This is swapping the first and last blocks in the list.
	batchedBlocks[0], batchedBlocks[batchSize-1] = batchedBlocks[batchSize-1], batchedBlocks[0]

	msg := p2p.Message{
		Ctx: context.Background(),
		Data: &pb.BatchedBeaconBlockResponse{
			BatchedBlocks: batchedBlocks,
		},
	}

	chainHead := &pb.ChainHeadResponse{}

	ss.processBatchedBlocks(msg, chainHead)
}

func TestProcessingBlocks_SkippedSlots(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	b, s := setUpGenesisStateAndBlock(db, t)
	ctx := context.Background()

	cfg := &Config{
		P2P:          &mockP2P{},
		SyncService:  &mockSyncService{},
		ChainService: &mockChainService{headBlock: b, headState: s},
		BeaconDB:     db,
	}
	ss := NewInitialSyncService(context.Background(), cfg)

	batchSize := 20
	blks, err := ss.db.BlocksBySlot(ctx, 0)
	if err != nil {
		t.Fatalf("Unable to get genesis block %v", err)
	}
	h, err := ssz.SigningRoot(blks[0])
	if err != nil {
		t.Fatalf("Unable to hash block %v", err)
	}
	parentHash := h[:]

	for i := 1; i <= batchSize; i++ {
		// skip slots
		if i == 4 || i == 6 || i == 13 || i == 17 {
			continue
		}
		block := &ethpb.BeaconBlock{
			Slot:       uint64(i),
			ParentRoot: parentHash,
		}

		chainHead := &pb.ChainHeadResponse{}

		ss.processBlock(context.Background(), block, chainHead)

		// Save the block and set the parent hash of the next block
		// as the hash of the current block.
		if err := ss.db.SaveBlock(block); err != nil {
			t.Fatalf("Block unable to be saved %v", err)
		}

		hash, err := ssz.SigningRoot(block)
		if err != nil {
			t.Fatalf("Could not hash block %v", err)
		}
		parentHash = hash[:]

	}

}

func TestSafelyHandleMessage(t *testing.T) {
	hook := logTest.NewGlobal()

	safelyHandleMessage(func(_ p2p.Message) error {
		panic("bad!")
	}, p2p.Message{
		Data: &ethpb.BeaconBlock{},
	})

	testutil.AssertLogsContain(t, hook, "Panicked when handling p2p message!")
}

func TestSafelyHandleMessage_NoData(t *testing.T) {
	hook := logTest.NewGlobal()

	safelyHandleMessage(func(_ p2p.Message) error {
		panic("bad!")
	}, p2p.Message{})

	entry := hook.LastEntry()
	if entry.Data["msg"] != "message contains no data" {
		t.Errorf("Message logged was not what was expected: %s", entry.Data["msg"])
	}
}
