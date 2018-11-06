package sync

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	btestutil "github.com/prysmaticlabs/prysm/beacon-chain/testutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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

func setupService(t *testing.T, db *db.BeaconDB) *Service {
	cfg := Config{
		BlockHashBufferSize: 0,
		BlockBufferSize:     0,
		ChainService:        &mockChainService{},
		P2P:                 &mockP2P{},
		BeaconDB:            db,
	}
	return NewSyncService(context.Background(), cfg)
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{
		BlockHashBufferSize: 0,
		BlockBufferSize:     0,
		ChainService:        &mockChainService{},
		P2P:                 &mockP2P{},
		BeaconDB:            db,
	}
	ss := NewSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	announceHash := hashutil.Hash([]byte{})
	hashAnnounce := &pb.BeaconBlockHashAnnounce{
		Hash: announceHash[:],
	}

	msg := p2p.Message{
		Ctx:  context.Background(),
		Peer: p2p.Peer{},
		Data: hashAnnounce,
	}

	// if a new hash is processed
	ss.announceBlockHashBuf <- msg

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	if err := db.InitializeState(nil); err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	if err := db.InitializeState(nil); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		BlockHashBufferSize: 0,
		BlockBufferSize:     0,
		ChainService:        &mockChainService{},
		P2P:                 &mockP2P{},
		BeaconDB:            db,
		AttestService:       &mockAttestService{},
	}
	ss := NewSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	parentBlock := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("failed to get parent hash: %v", err)
	}

	data := &pb.BeaconBlock{
		PowChainRef:    []byte{1, 2, 3, 4, 5},
		AncestorHashes: [][]byte{parentHash[:]},
		Slot:           1,
	}
	attestation := &pb.AggregatedAttestation{
		Slot:           0,
		Shard:          0,
		ShardBlockHash: []byte{'A'},
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

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)

	if err := db.InitializeState(nil); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		BlockHashBufferSize: 0,
		BlockBufferSize:     0,
		ChainService:        &mockChainService{},
		P2P:                 &mockP2P{},
		BeaconDB:            db,
		AttestService:       &mockAttestService{},
	}
	ss := NewSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	parentBlock := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	if err := db.SaveBlock(parentBlock); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("failed to get parent hash: %v", err)
	}

	data1 := &pb.BeaconBlock{
		PowChainRef:    []byte{1, 2, 3, 4, 5},
		AncestorHashes: [][]byte{parentHash[:]},
		Slot:           1,
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block:       data1,
		Attestation: &pb.AggregatedAttestation{},
	}

	msg1 := p2p.Message{
		Ctx:  context.Background(),
		Peer: p2p.Peer{},
		Data: responseBlock1,
	}

	data2 := &pb.BeaconBlock{
		PowChainRef:    []byte{6, 7, 8, 9, 10},
		AncestorHashes: [][]byte{make([]byte, 32)},
		Slot:           1,
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block:       data2,
		Attestation: &pb.AggregatedAttestation{},
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

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
	ss := setupService(t, db)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		<-exitRoutine
	}()

	malformedRequest := &pb.BeaconBlockHashAnnounce{
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

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)
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

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)

	cfg := Config{
		BlockHashBufferSize:    0,
		BlockBufferSize:        0,
		BlockRequestBufferSize: 0,
		ChainService:           ms,
		AttestService:          as,
		P2P:                    &mockP2P{},
		BeaconDB:               db,
	}
	ss := NewSyncService(context.Background(), cfg)

	exitRoutine := make(chan bool)
	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.AggregatedAttestation{
		Slot:             0,
		AttesterBitfield: []byte{99},
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

func TestStartNotSynced(t *testing.T) {
	hook := logTest.NewGlobal()

	db := btestutil.SetupDB(t)
	defer btestutil.TeardownDB(t, db)

	cfg := DefaultConfig()
	cfg.ChainService = &mockChainService{}
	cfg.P2P = &mockP2P{}
	cfg.BeaconDB = db
	ss := NewSyncService(context.Background(), cfg)

	ss.Start()
	ss.Stop()

	testutil.AssertLogsContain(t, hook, "Not caught up with network, but continue sync")
	testutil.AssertLogsContain(t, hook, "Stopping service")

	hook.Reset()
}
