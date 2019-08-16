package sync

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/prysmaticlabs/go-ssz"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	p2p "github.com/prysmaticlabs/prysm/shared/deprecated-p2p"
	"github.com/prysmaticlabs/prysm/shared/event"
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

type mockP2P struct {
	sentMsg proto.Message
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(ctx context.Context, msg proto.Message) {
}

func (mp *mockP2P) Send(ctx context.Context, msg proto.Message, peerID peer.ID) error {
	mp.sentMsg = msg
	return nil
}

func (mp *mockP2P) Reputation(_ peer.ID, val int) {

}

type mockChainService struct {
	sFeed *event.Feed
	cFeed *event.Feed
	db    *db.BeaconDB
}

func (ms *mockChainService) StateInitializedFeed() *event.Feed {
	if ms.sFeed == nil {
		return new(event.Feed)
	}
	return ms.sFeed
}

func (ms *mockChainService) CanonicalBlockFeed() *event.Feed {
	if ms.cFeed == nil {
		return new(event.Feed)
	}
	return ms.cFeed
}

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *ethpb.BeaconBlock) (*pb.BeaconState, error) {
	if err := ms.db.SaveBlock(block); err != nil {
		return nil, err
	}
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) AdvanceState(
	ctx context.Context, beaconState *pb.BeaconState, block *ethpb.BeaconBlock,
) (*pb.BeaconState, error) {
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) VerifyBlockValidity(ctx context.Context, block *ethpb.BeaconBlock, beaconState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) ApplyForkChoiceRule(ctx context.Context, block *ethpb.BeaconBlock, computedState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) CleanupBlockOperations(ctx context.Context, block *ethpb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) IsCanonical(slot uint64, hash []byte) bool {
	return true
}

func (ms *mockChainService) UpdateCanonicalRoots(block *ethpb.BeaconBlock, root [32]byte) {
}

type mockOperationService struct{}

func (ms *mockOperationService) IncomingProcessedBlockFeed() *event.Feed {
	return nil
}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingExitFeed() *event.Feed {
	return new(event.Feed)
}

type mockAttestationService struct{}

func (ma *mockAttestationService) IncomingAttestationFeed() *event.Feed {
	return new(event.Feed)
}

func setupService(db *db.BeaconDB) *RegularSync {
	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
	}
	return NewRegularSyncService(context.Background(), cfg)
}

func TestProcessBlockRoot_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	announceHash := hashutil.Hash([]byte{})
	hashAnnounce := &pb.BeaconBlockAnnounce{
		Hash: announceHash[:],
	}

	msg := p2p.Message{
		Ctx:  context.Background(),
		Peer: "",
		Data: hashAnnounce,
	}

	// if a new hash is processed
	if err := ss.receiveBlockAnnounce(msg); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	validators := make([]*ethpb.Validator, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService: &mockChainService{
			db: db,
		},
		P2P:              &mockP2P{},
		BeaconDB:         db,
		OperationService: &mockOperationService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	parentBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent root: %v", err)
	}

	data := &ethpb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       0,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{1, 2, 3, 4, 5},
				BlockHash:   []byte{6, 7, 8, 9, 10},
			},
		},
	}
	attestation := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard:    0,
				DataRoot: []byte{'A'},
			},
		},
	}

	responseBlock := &pb.BeaconBlockResponse{
		Block:       data,
		Attestation: attestation,
	}

	msg := p2p.Message{
		Ctx:  context.Background(),
		Peer: "",
		Data: responseBlock,
	}

	if err := ss.receiveBlock(msg); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "Sending newly received block to chain service")
	hook.Reset()
}

func TestProcessBlock_MultipleBlocksProcessedOK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)

	validators := make([]*ethpb.Validator, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatal(err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService: &mockChainService{
			db: db,
		},
		P2P:              &mockP2P{},
		BeaconDB:         db,
		OperationService: &mockOperationService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	parentBlock := &ethpb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentRoot, err := ssz.SigningRoot(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent root: %v", err)
	}

	data1 := &ethpb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       1,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{1, 2, 3, 4, 5},
				BlockHash:   []byte{6, 7, 8, 9, 10},
			},
		},
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block: data1,
		Attestation: &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    0,
					DataRoot: []byte{},
				},
			},
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Peer: "",
		Data: responseBlock1,
	}

	data2 := &ethpb.BeaconBlock{
		ParentRoot: []byte{},
		Slot:       1,
		Body: &ethpb.BeaconBlockBody{
			Eth1Data: &ethpb.Eth1Data{
				DepositRoot: []byte{11, 12, 13, 14, 15},
				BlockHash:   []byte{16, 17, 18, 19, 20},
			},
		},
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block: data2,
		Attestation: &ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard:    0,
					DataRoot: []byte{},
				},
			},
		},
	}

	msg2 := p2p.Message{
		Ctx:  context.Background(),
		Peer: "",
		Data: responseBlock2,
	}

	if err := ss.receiveBlock(msg1); err != nil {
		t.Error(err)
	}
	if err := ss.receiveBlock(msg2); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "Sending newly received block to chain service")
	testutil.AssertLogsContain(t, hook, "Sending newly received block to chain service")
	hook.Reset()
}

func TestReceiveAttestation_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}
	ctx := context.Background()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	beaconState := &pb.BeaconState{
		Slot:                2,
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	beaconBlock := &ethpb.BeaconBlock{
		Slot: beaconState.Slot,
	}
	if err := db.SaveBlock(beaconBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateChainHead(ctx, beaconBlock, beaconState); err != nil {
		t.Fatal(err)
	}
	cfg := &RegularSyncConfig{
		ChainService:     ms,
		AttsService:      &mockAttestationService{},
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	request1 := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 1,
			},
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}

	if err := ss.receiveAttestation(msg1); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "Sending newly received attestation to subscribers")
}

func TestReceiveAttestation_OlderThanPrevEpoch(t *testing.T) {
	helpers.ClearAllCaches()

	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}
	ctx := context.Background()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	state := &pb.BeaconState{
		Slot:                2 * params.BeaconConfig().SlotsPerEpoch,
		FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 1},
	}
	if err := db.SaveState(ctx, state); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	headBlock := &ethpb.BeaconBlock{Slot: state.Slot}
	if err := db.SaveBlock(headBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(ctx, headBlock, state); err != nil {
		t.Fatalf("failed to update chain head: %v", err)
	}
	cfg := &RegularSyncConfig{
		AttsService:      &mockAttestationService{},
		ChainService:     ms,
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	request1 := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 900,
			},
			Source: &ethpb.Checkpoint{},
			Target: &ethpb.Checkpoint{},
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}

	if err := ss.receiveAttestation(msg1); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, "Skipping received attestation with target epoch less than current finalized epoch")
}

func TestReceiveExitReq_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	request1 := &ethpb.VoluntaryExit{
		Epoch: 100,
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}
	if err := ss.receiveExitRequest(msg1); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "Forwarding validator exit request to subscribed services")
}

func TestHandleStateReq_NOState(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)

	ss := setupService(db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	request1 := &pb.BeaconStateRequest{
		FinalizedStateRootHash32S: []byte{'a'},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}

	if err := ss.handleStateRequest(msg1); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, "Requested state root is diff than local state root")

}

func TestHandleStateReq_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ctx := context.Background()
	helpers.ClearAllCaches()

	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := db.InitializeState(context.Background(), unixTime, []*ethpb.Deposit{}, &ethpb.Eth1Data{}); err != nil {
		t.Fatalf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("could not attempt fetch beacon state: %v", err)
	}
	if err := db.SaveJustifiedState(beaconState); err != nil {
		t.Fatalf("could not save justified state: %v", err)
	}
	if err := db.SaveFinalizedState(beaconState); err != nil {
		t.Fatalf("could not save justified state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("could not hash beacon state: %v", err)
	}
	genBlock := b.NewGenesisBlock(stateRoot[:])
	if err := db.SaveFinalizedBlock(genBlock); err != nil {
		t.Fatalf("could not save genesis block: %v", err)
	}

	ss := setupService(db)

	request1 := &pb.BeaconStateRequest{
		FinalizedStateRootHash32S: stateRoot[:],
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}

	if err := ss.handleStateRequest(msg1); err != nil {
		t.Error(err)
	}
	testutil.AssertLogsContain(t, hook, "Sending finalized state and block to peer")
}

func TestCanonicalBlockList_CanRetrieveCanonical(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ss := setupService(db)

	// Construct the following chain:
	//    /- B3
	//	 B1  - B2 - B4
	block1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}
	root1, err := ssz.SigningRoot(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = ss.db.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: root1[:]}
	root2, _ := ssz.SigningRoot(block2)
	if err = ss.db.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: root1[:]}
	if err = ss.db.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: root2[:]}
	root4, _ := ssz.SigningRoot(block4)
	if err = ss.db.SaveBlock(block4); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}

	// Verify passing in roots of B4 and B1 give us the canonical lists.
	list, err := ss.respondBatchedBlocks(context.Background(), root1[:], root4[:])
	if err != nil {
		t.Fatal(err)
	}
	wantList := []*ethpb.BeaconBlock{block2, block4}
	if !reflect.DeepEqual(list, wantList) {
		t.Error("Did not retrieve the correct canonical lists")
	}
}

func TestCanonicalBlockList_SameFinalizedAndHead(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ss := setupService(db)

	// Construct the following chain:
	//	 B1 (finalized and head)
	block1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'A'}}
	root1, err := ssz.SigningRoot(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = ss.db.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}

	// Verify passing in roots of B1 and B1 give us the canonical lists which should be an empty list.
	list, err := ss.respondBatchedBlocks(context.Background(), root1[:], root1[:])
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Error("Did not retrieve the correct canonical lists")
	}
}

func TestCanonicalBlockList_NilBlock(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ss := setupService(db)

	want := "nil block 0x42 from db"
	if _, err := ss.respondBatchedBlocks(context.Background(), []byte{'A'}, []byte{'B'}); err.Error() != want {
		t.Fatal(err)
	}
}

func TestCanonicalBlockList_NilParentBlock(t *testing.T) {
	db := internal.SetupDBDeprecated(t)
	defer internal.TeardownDBDeprecated(t, db)
	ss := setupService(db)

	block1 := &ethpb.BeaconBlock{Slot: 1, ParentRoot: []byte{'B'}}
	root1, err := ssz.SigningRoot(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = ss.db.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}

	want := fmt.Sprintf("nil parent block %#x from db", []byte{'B'})
	if _, err := ss.respondBatchedBlocks(context.Background(), []byte{}, root1[:]); err.Error() != want {
		t.Log(want)
		t.Fatal(err)
	}
}
