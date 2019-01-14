package sync

import (
	"context"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

type mockChainService struct{}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

type mockAttestService struct{}

func (ms *mockAttestService) IncomingAttestationFeed() *event.Feed {
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

func TestProcessBlockHash(t *testing.T) {
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

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			Balance: params.BeaconConfig().MaxDepositInGwei,
			Pubkey:  []byte(strconv.Itoa(i)),
			RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
				169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
		}
	}
	if err := db.InitializeState(); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
		AttestService:           &mockAttestService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	parentBlock := &pb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentHash, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent hash: %v", err)
	}

	data := &pb.BeaconBlock{
		DepositRootHash32: []byte{1, 2, 3, 4, 5},
		ParentRootHash32:  parentHash[:],
		Slot:              1,
	}
	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			Slot:                 0,
			Shard:                0,
			ShardBlockRootHash32: []byte{'A'},
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

func TestProcessMultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			Balance: params.BeaconConfig().MaxDepositInGwei,
			Pubkey:  []byte(strconv.Itoa(i)),
			RandaoCommitmentHash32: []byte{41, 13, 236, 217, 84, 139, 98, 168, 214, 3, 69,
				169, 136, 56, 111, 200, 75, 166, 188, 149, 72, 64, 8, 246, 54, 47, 147, 22, 14, 243, 229, 99},
		}
	}
	if err := db.InitializeState(); err != nil {
		t.Fatal(err)
	}

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		ChainService:            &mockChainService{},
		P2P:                     &mockP2P{},
		BeaconDB:                db,
		AttestService:           &mockAttestService{},
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	parentBlock := &pb.BeaconBlock{
		Slot: 0,
	}
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentHash, err := hashutil.HashBeaconBlock(parentBlock)
	if err != nil {
		t.Fatalf("failed to get parent hash: %v", err)
	}

	data1 := &pb.BeaconBlock{
		DepositRootHash32: []byte{1, 2, 3, 4, 5},
		ParentRootHash32:  parentHash[:],
		Slot:              1,
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block: data1,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				ShardBlockRootHash32: []byte{},
				Slot:                 0,
				JustifiedSlot:        0,
			},
		},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Peer: p2p.Peer{},
		Data: responseBlock1,
	}

	data2 := &pb.BeaconBlock{
		DepositRootHash32: []byte{6, 7, 8, 9, 10},
		ParentRootHash32:  []byte{},
		Slot:              1,
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block: data2,
		Attestation: &pb.Attestation{
			Data: &pb.AttestationData{
				ShardBlockRootHash32: []byte{},
				Slot:                 0,
				JustifiedSlot:        0,
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

func TestBlockRequestErrors(t *testing.T) {
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

func TestBlockRequest(t *testing.T) {
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

func TestReceiveAttestation(t *testing.T) {
	hook := logTest.NewGlobal()
	ms := &mockChainService{}
	as := &mockAttestService{}

	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	cfg := &RegularSyncConfig{
		BlockAnnounceBufferSize: 0,
		BlockBufferSize:         0,
		BlockReqHashBufferSize:  0,
		BlockReqSlotBufferSize:  0,
		ChainService:            ms,
		AttestService:           as,
		P2P:                     &mockP2P{},
		BeaconDB:                db,
	}
	ss := NewRegularSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.Attestation{
		ParticipationBitfield: []byte{99},
		Data: &pb.AttestationData{
			Slot: 0,
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
	testutil.AssertLogsContain(t, hook, "Forwarding attestation to subscribed services")
}
