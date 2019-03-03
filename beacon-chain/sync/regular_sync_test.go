package sync

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
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
}

func (mp *mockP2P) Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg proto.Message) {}

func (mp *mockP2P) Send(msg proto.Message, peer p2p.Peer) {
}

type mockChainService struct {
	bFeed *event.Feed
	sFeed *event.Feed
}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	if ms.bFeed == nil {
		return new(event.Feed)
	}
	return ms.bFeed
}

func (ms *mockChainService) StateInitializedFeed() *event.Feed {
	if ms.sFeed == nil {
		return new(event.Feed)
	}
	return ms.sFeed
}

type mockOperationService struct{}

func (ms *mockOperationService) IncomingAttFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockOperationService) IncomingExitFeed() *event.Feed {
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

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	announceHash := hashutil.Hash([]byte{})
	hashAnnounce := &pb.BeaconBlockAnnounce{
		Hash: announceHash[:],
	}

	msg := p2p.Message{
		Ctx:  context.Background(),
		Peer: p2p.Peer{},
		Data: hashAnnounce,
	}

	// if a new hash is processed
	ss.announceBlockBuf <- msg

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			Pubkey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
		OperationService:        &mockOperationService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

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
		Peer: p2p.Peer{},
		Data: responseBlock,
	}

	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	hook.Reset()
}

func TestProcessBlock_MultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			Pubkey: []byte(strconv.Itoa(i)),
		}
	}
	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits); err != nil {
		t.Fatal(err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
		OperationService:        &mockOperationService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

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
		Peer: p2p.Peer{},
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
		Peer: p2p.Peer{},
		Data: responseBlock2,
	}

	ss.blockBuf <- msg1
	ss.blockBuf <- msg2
	ss.cancel()
	<-exitRoutine
	testutil.AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	testutil.AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	hook.Reset()
}

func TestBlockRequest_InvalidMsg(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(t, db)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		<-exitRoutine
	}()

	malformedRequest := &pb.BeaconBlockAnnounce{
		Hash: []byte{'t', 'e', 's', 't'},
	}

	invalidmsg := p2p.Message{
		Ctx:  context.Background(),
		Data: malformedRequest,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- invalidmsg
	ss.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Received malformed beacon block request p2p message")
}

func TestBlockRequest_OK(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ss := setupService(t, db)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		<-exitRoutine
	}()

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- msg1
	ss.cancel()
	exitRoutine <- true

	testutil.AssertLogsDoNotContain(t, hook, "Sending requested block to peer")
}

func TestReceiveAttestation_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	if err := db.SaveState(&pb.BeaconState{
		FinalizedEpoch: params.BeaconConfig().GenesisEpoch,
	}); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	cfg := &RegularSyncConfig{
		ChainService:     ms,
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot: params.BeaconConfig().GenesisSlot + 1,
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.attestationBuf <- msg1
	ss.cancel()
	<-exitRoutine
	testutil.AssertLogsContain(t, hook, "Sending newly received attestation to subscribers")
}

func TestReceiveAttestation_OlderThanFinalizedEpoch(t *testing.T) {
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	os := &mockOperationService{}

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	state := &pb.BeaconState{FinalizedEpoch: params.BeaconConfig().GenesisEpoch + 1}
	if err := db.SaveState(state); err != nil {
		t.Fatalf("Could not save state: %v", err)
	}
	cfg := &RegularSyncConfig{
		ChainService:     ms,
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot: params.BeaconConfig().GenesisSlot + 1,
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.attestationBuf <- msg1
	ss.cancel()
	<-exitRoutine
	want := fmt.Sprintf(
		"Skipping received attestation with slot smaller than last finalized slot, %d < %d",
		request1.Data.Slot, state.FinalizedEpoch*params.BeaconConfig().SlotsPerEpoch)
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
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.VoluntaryExit{
		Epoch: 100,
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.exitBuf <- msg1
	ss.cancel()
	<-exitRoutine
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
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	req := &pb.AttestationRequest{
		Hash: []byte{'A'},
	}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: p2p.Peer{},
	}

	ss.attestationReqByHashBuf <- msg
	ss.cancel()
	<-exitRoutine
	want := fmt.Sprintf("Attestation %#x is not in db", bytesutil.ToBytes32(req.Hash))
	testutil.AssertLogsContain(t, hook, want)
}

func TestHandleUnseenAttsReq_EmptyAttsPool(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	req := &pb.UnseenAttestationsRequest{}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: p2p.Peer{},
	}

	ss.unseenAttestationsReqBuf <- msg
	ss.cancel()
	<-exitRoutine
	testutil.AssertLogsContain(t, hook, "There's no unseen attestation in db")
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
	if err := db.SaveAttestation(att); err != nil {
		t.Fatalf("Could not save attestation: %v", err)
	}

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	req := &pb.AttestationRequest{
		Hash: attRoot[:],
	}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: p2p.Peer{},
	}

	ss.attestationReqByHashBuf <- msg
	ss.cancel()
	<-exitRoutine
	want := fmt.Sprintf("Sending attestation %#x to peer", attRoot)
	testutil.AssertLogsContain(t, hook, want)
}

func TestHandleUnseenAttsReq_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	os := &mockOperationService{}
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	att := &pb.Attestation{
		AggregationBitfield: []byte{'D', 'E', 'F'},
	}
	if err := db.SaveAttestation(att); err != nil {
		t.Fatalf("Could not save attestation: %v", err)
	}

	cfg := &RegularSyncConfig{
		OperationService: os,
		P2P:              &mockP2P{},
		BeaconDB:         db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	req := &pb.UnseenAttestationsRequest{}
	msg := p2p.Message{
		Ctx:  context.Background(),
		Data: req,
		Peer: p2p.Peer{},
	}

	ss.unseenAttestationsReqBuf <- msg
	ss.cancel()
	<-exitRoutine
	testutil.AssertLogsContain(t, hook, "Sending response for batched unseen attestations to peer")
}

func TestHandleStateReq_NOState(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	ss := setupService(t, db)

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	if err := db.InitializeState(genesisTime, deposits); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		<-exitRoutine
	}()

	request1 := &pb.BeaconStateRequest{
		Hash: []byte{'a'},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.stateRequestBuf <- msg1

	ss.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Requested state root is different from locally stored state root")

}

func TestHandleStateReq_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	genesisTime := time.Now()
	unixTime := uint64(genesisTime.Unix())
	if err := db.InitializeState(unixTime, []*pb.Deposit{}); err != nil {
		t.Fatalf("could not initialize beacon state to disk: %v", err)
	}
	beaconState, err := db.State(ctx)
	if err != nil {
		t.Fatalf("could not attempt fetch beacon state: %v", err)
	}
	stateRoot, err := hashutil.HashProto(beaconState)
	if err != nil {
		t.Fatalf("could not hash beacon state: %v", err)
	}

	ss := setupService(t, db)
	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		<-exitRoutine
	}()

	request1 := &pb.BeaconStateRequest{
		Hash: stateRoot[:],
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.stateRequestBuf <- msg1

	ss.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Sending beacon state to peer")
}
