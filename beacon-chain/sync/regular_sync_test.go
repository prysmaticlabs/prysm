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
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
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

func (ms *mockChainService) ReceiveBlock(ctx context.Context, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	if err := ms.db.SaveBlock(block); err != nil {
		return nil, err
	}
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) AdvanceState(
	ctx context.Context, beaconState *pb.BeaconState, block *pb.BeaconBlock,
) (*pb.BeaconState, error) {
	return &pb.BeaconState{}, nil
}

func (ms *mockChainService) VerifyBlockValidity(ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) ApplyForkChoiceRule(ctx context.Context, block *pb.BeaconBlock, computedState *pb.BeaconState) error {
	return nil
}

func (ms *mockChainService) CleanupBlockOperations(ctx context.Context, block *pb.BeaconBlock) error {
	return nil
}

func (ms *mockChainService) IsCanonical(slot uint64, hash []byte) bool {
	return true
}

func (ms *mockChainService) UpdateCanonicalRoots(block *pb.BeaconBlock, root [32]byte) {
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

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

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

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	validators := make([]*pb.Validator, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			Pubkey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{}); err != nil {
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

	parentBlock := &pb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentRoot, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent root: %v", err)
	}

	data := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{1, 2, 3, 4, 5},
			BlockRoot:   []byte{6, 7, 8, 9, 10},
		},
		ParentBlockRoot: parentRoot[:],
		Slot:            0,
	}
	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:              0,
			Shard:             0,
			CrosslinkDataRoot: []byte{'A'},
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

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	validators := make([]*pb.Validator, 10)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			Pubkey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{}); err != nil {
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

	parentBlock := &pb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentRoot, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent root: %v", err)
	}

	data1 := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{1, 2, 3, 4, 5},
			BlockRoot:   []byte{6, 7, 8, 9, 10},
		},
		ParentBlockRoot: parentRoot[:],
		Slot:            1,
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block: data1,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				CrosslinkDataRoot: []byte{},
				Slot:              0,
			},
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Peer: "",
		Data: responseBlock1,
	}

	data2 := &pb.BeaconBlock{
		Eth1Data: &pb.Eth1Data{
			DepositRoot: []byte{11, 12, 13, 14, 15},
			BlockRoot:   []byte{16, 17, 18, 19, 20},
		},
		ParentBlockRoot: []byte{},
		Slot:            1,
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block: data2,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				CrosslinkDataRoot: []byte{},
				Slot:              0,
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

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	beaconState := &pb.BeaconState{
		Slot: 2,
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	beaconBlock := &pb.BeaconBlock{
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

	request1 := &pb.AttestationResponse{
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				Slot: 1,
			},
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
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}
	ctx := context.Background()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	state := &pb.BeaconState{Slot: 2 * params.BeaconConfig().SlotsPerEpoch}
	if err := db.SaveState(ctx, state); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	headBlock := &pb.BeaconBlock{Slot: state.Slot}
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

	request1 := &pb.AttestationResponse{
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				Slot: 0,
			},
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

	testutil.AssertLogsContain(t, hook, "Skipping received attestation with slot smaller than one epoch ago")
}

func TestReceiveExitReq_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	request1 := &pb.VoluntaryExit{
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

func TestHandleAttReq_HashNotFound(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	req := &pb.AttestationRequest{
		Hash: []byte{'A'},
	}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: "",
	}

	if err := ss.handleAttestationRequestByHash(msg); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, "Attestation not in db")
}

func TestHandleAnnounceAttestation_requestsAttestationData(t *testing.T) {
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	att := &pb.Attestation{
		AggregationBitfield: []byte{'A', 'B', 'C'},
	}
	hash, err := hashutil.HashProto(att)
	if err != nil {
		t.Fatalf("Could not hash attestation: %v", err)
	}
	sender := &mockP2P{}
	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              sender,
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	if err := ss.handleAttestationAnnouncement(p2p.Message{
		Ctx:  context.Background(),
		Data: &pb.AttestationAnnounce{Hash: hash[:]},
	}); err != nil {
		t.Fatal(err)
	}

	if sender.sentMsg == nil {
		t.Fatal("send was not called")
	}

	msg, ok := sender.sentMsg.(*pb.AttestationRequest)
	if !ok {
		t.Fatal("sent p2p message is wrong type")
	}

	if bytesutil.ToBytes32(msg.Hash) != hash {
		t.Fatal("message didnt include the proper hash")
	}
}

func TestHandleAnnounceAttestation_doNothingIfAlreadySeen(t *testing.T) {
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	att := &pb.Attestation{
		AggregationBitfield: []byte{'A', 'B', 'C'},
	}
	hash, err := hashutil.HashProto(att)
	if err != nil {
		t.Fatalf("Could not hash attestation: %v", err)
	}
	if err := db.SaveAttestation(context.Background(), att); err != nil {
		t.Fatalf("Could not save attestation: %v", err)
	}
	sender := &mockP2P{}
	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              sender,
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	if err := ss.handleAttestationAnnouncement(p2p.Message{
		Ctx:  context.Background(),
		Data: &pb.AttestationAnnounce{Hash: hash[:]},
	}); err != nil {
		t.Error(err)
	}

	if sender.sentMsg != nil {
		t.Error("send was called, but it should not have been called")
	}

}

func TestHandleAttReq_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	att := &pb.Attestation{
		AggregationBitfield: []byte{'A', 'B', 'C'},
	}
	attRoot, err := hashutil.HashProto(att)
	if err != nil {
		t.Fatalf("Could not hash attestation: %v", err)
	}
	if err := db.SaveAttestation(context.Background(), att); err != nil {
		t.Fatalf("Could not save attestation: %v", err)
	}

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
		ChainService:     &mockChainService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	req := &pb.AttestationRequest{
		Hash: attRoot[:],
	}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: "",
	}

	if err := ss.handleAttestationRequestByHash(msg); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, "Sending attestation to peer")
}

func TestHandleStateReq_NOState(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	ss := setupService(db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t)
	if err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{}); err != nil {
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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := db.InitializeState(context.Background(), unixTime, []*pb.Deposit{}, &pb.Eth1Data{}); err != nil {
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
	testutil.AssertLogsContain(t, hook, "Sending finalized, justified, and canonical states to peer")
}

func TestCanonicalBlockList_CanRetrieveCanonical(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(db)

	// Construct the following chain:
	//    /- B3
	//	 B1  - B2 - B4
	block1 := &pb.BeaconBlock{Slot: 1, ParentBlockRoot: []byte{'A'}}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = ss.db.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block2 := &pb.BeaconBlock{Slot: 2, ParentBlockRoot: root1[:]}
	root2, _ := hashutil.HashBeaconBlock(block2)
	if err = ss.db.SaveBlock(block2); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block3 := &pb.BeaconBlock{Slot: 3, ParentBlockRoot: root1[:]}
	if err = ss.db.SaveBlock(block3); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}
	block4 := &pb.BeaconBlock{Slot: 4, ParentBlockRoot: root2[:]}
	root4, _ := hashutil.HashBeaconBlock(block4)
	if err = ss.db.SaveBlock(block4); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}

	// Verify passing in roots of B4 and B1 give us the canonical lists.
	list, err := ss.respondBatchedBlocks(context.Background(), root1[:], root4[:])
	wantList := []*pb.BeaconBlock{block2, block4}
	if !reflect.DeepEqual(list, wantList) {
		t.Error("Did not retrieve the correct canonical lists")
	}
}

func TestCanonicalBlockList_SameFinalizedAndHead(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(db)

	// Construct the following chain:
	//	 B1 (finalized and head)
	block1 := &pb.BeaconBlock{Slot: 1, ParentBlockRoot: []byte{'A'}}
	root1, err := hashutil.HashBeaconBlock(block1)
	if err != nil {
		t.Fatalf("Could not hash block: %v", err)
	}
	if err = ss.db.SaveBlock(block1); err != nil {
		t.Fatalf("Could not save block: %v", err)
	}

	// Verify passing in roots of B1 and B1 give us the canonical lists which should be an empty list.
	list, err := ss.respondBatchedBlocks(context.Background(), root1[:], root1[:])
	if len(list) != 0 {
		t.Error("Did not retrieve the correct canonical lists")
	}
}

func TestCanonicalBlockList_NilBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(db)

	want := "nil block 0x42 from db"
	if _, err := ss.respondBatchedBlocks(context.Background(), []byte{'A'}, []byte{'B'}); err.Error() != want {
		t.Fatal(err)
	}
}

func TestCanonicalBlockList_NilParentBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(db)

	block1 := &pb.BeaconBlock{Slot: 1, ParentBlockRoot: []byte{'B'}}
	root1, err := hashutil.HashBeaconBlock(block1)
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
