package sync

import (
	"context"
	"fmt"
	"io/ioutil"
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

type mockChainService struct {
	bFeed *event.Feed
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

func (ms *mockChainService) ApplyBlockStateTransition(ctx context.Context, block *pb.BeaconBlock, beaconState *pb.BeaconState) (*pb.BeaconState, error) {
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

func setupService(t *testing.T, db *db.BeaconDB) *RegularSync {
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
	deposits, _ := setupInitialDeposits(t, 10)
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
		Slot: params.BeaconConfig().GenesisSlot,
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
			DepositRootHash32: []byte{1, 2, 3, 4, 5},
			BlockHash32:       []byte{6, 7, 8, 9, 10},
		},
		ParentRootHash32: parentRoot[:],
		Slot:             params.BeaconConfig().GenesisSlot,
	}
	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:                    0,
			Shard:                   0,
			CrosslinkDataRootHash32: []byte{'A'},
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
	deposits, _ := setupInitialDeposits(t, 10)
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
		Slot: params.BeaconConfig().GenesisSlot,
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
			DepositRootHash32: []byte{1, 2, 3, 4, 5},
			BlockHash32:       []byte{6, 7, 8, 9, 10},
		},
		ParentRootHash32: parentRoot[:],
		Slot:             params.BeaconConfig().GenesisSlot + 1,
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block: data1,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				CrosslinkDataRootHash32: []byte{},
				Slot:                    params.BeaconConfig().GenesisSlot,
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
			DepositRootHash32: []byte{11, 12, 13, 14, 15},
			BlockHash32:       []byte{16, 17, 18, 19, 20},
		},
		ParentRootHash32: []byte{},
		Slot:             1,
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block: data2,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				CrosslinkDataRootHash32: []byte{},
				Slot:                    0,
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

func TestBlockRequest_InvalidMsg(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(t, db)

	malformedRequest := &pb.BeaconBlockAnnounce{
		Hash: []byte{'t', 'e', 's', 't'},
	}

	invalidmsg := p2p.Message{
		Ctx:  context.Background(),
		Data: malformedRequest,
		Peer: "",
	}

	if err := ss.handleBlockRequestBySlot(invalidmsg); err == nil {
		t.Error("Expected error, received nil")
	}
	testutil.AssertLogsContain(t, hook, "Received malformed beacon block request p2p message")
}

func TestBlockRequest_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(t, db)

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: "",
	}
	if err := db.SaveBlock(&pb.BeaconBlock{Slot: 20}); err != nil {
		t.Fatal(err)
	}

	if err := ss.handleBlockRequestBySlot(msg1); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, "Sending requested block to peer")
}

func TestReceiveAttestation_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}
	ctx := context.Background()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	if err := db.SaveState(ctx, &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 2,
	}); err != nil {
		t.Fatalf("Could not save state: %v", err)
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
				Slot: params.BeaconConfig().GenesisSlot + 1,
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
	state := &pb.BeaconState{Slot: params.BeaconConfig().GenesisSlot + 2*params.BeaconConfig().SlotsPerEpoch}
	if err := db.SaveState(ctx, state); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	cfg := &RegularSyncConfig{
		ChainService:     ms,
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	request1 := &pb.AttestationResponse{
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				Slot: params.BeaconConfig().GenesisSlot,
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
	want := fmt.Sprintf(
		"Skipping received attestation with slot smaller than one epoch ago, %d < %d",
		request1.Attestation.Data.Slot, params.BeaconConfig().GenesisSlot+params.BeaconConfig().SlotsPerEpoch)
	testutil.AssertLogsContain(t, hook, want)
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
	want := fmt.Sprintf("Attestation %#x is not in db", bytesutil.ToBytes32(req.Hash))
	testutil.AssertLogsContain(t, hook, want)
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
	want := fmt.Sprintf("Sending attestation %#x to peer", attRoot)
	testutil.AssertLogsContain(t, hook, want)
}

func TestHandleStateReq_NOState(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	ss := setupService(t, db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
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

	testutil.AssertLogsContain(t, hook, "Requested state root is different from locally stored state root")

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

	ss := setupService(t, db)

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
