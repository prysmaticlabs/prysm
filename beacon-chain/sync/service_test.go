package sync

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	. "github.com/prysmaticlabs/prysm/beacon-chain/testutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	. "github.com/prysmaticlabs/prysm/shared/testutils"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/blake2b"
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

type mockChainService struct {}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) GetCanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error) {
	return types.NewBlock(&pb.BeaconBlock{SlotNumber: slotnumber}), nil
}

type mockDB struct {
	hasState bool
	getError bool
}

func (mdb *mockDB) HasInitialState() bool {
	return mdb.hasState
}

func (mdb *mockDB) HasBlock([32]byte) bool {
	return false
}

func (mdb *mockDB) HasBlockForSlot(uint64) bool {
	return true
}

func (mdb *mockDB) GetBlockBySlot(uint64) (*types.Block, error) {
	if mdb.getError {
		return nil, errors.New("mock get canonical block error")
	}
	return types.NewBlock(nil), nil
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, &mockChainService{}, SetupDB(t))

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	announceHash := blake2b.Sum512([]byte{})
	hashAnnounce := &pb.BeaconBlockHashAnnounce{
		Hash: announceHash[:],
	}

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: hashAnnounce,
	}

	// if a new hash is processed
	ss.announceBlockHashBuf <- msg

	ss.cancel()
	<-exitRoutine

	AssertLogsContain(t, hook, "requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, SetupDB(t))

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	data := &pb.BeaconBlock{
		PowChainRef: []byte{1, 2, 3, 4, 5},
		ParentHash:  make([]byte, 32),
	}

	responseBlock := &pb.BeaconBlockResponse{
		Block: data,
	}

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: responseBlock,
	}

	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	hook.Reset()
}

func TestProcessMultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, SetupDB(t))

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	data1 := &pb.BeaconBlock{
		PowChainRef: []byte{1, 2, 3, 4, 5},
		ParentHash:  make([]byte, 32),
	}

	responseBlock1 := &pb.BeaconBlockResponse{
		Block: data1,
	}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: responseBlock1,
	}

	data2 := &pb.BeaconBlock{
		PowChainRef: []byte{6, 7, 8, 9, 10},
		ParentHash:  make([]byte, 32),
	}

	responseBlock2 := &pb.BeaconBlockResponse{
		Block: data2,
	}

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: responseBlock2,
	}

	ss.blockBuf <- msg1
	ss.blockBuf <- msg2
	ss.cancel()
	<-exitRoutine
	AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	hook.Reset()
}

func TestBlockRequestErrors(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, SetupDB(t))

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	malformedRequest := &pb.BeaconBlockHashAnnounce{
		Hash: []byte{'t', 'e', 's', 't'},
	}

	invalidmsg := p2p.Message{
		Data: malformedRequest,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- invalidmsg
	AssertLogsContain(t, hook, "Received malformed beacon block request p2p message")

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- msg1
	AssertLogsDoNotContain(t, hook, "Sending requested block to peer")
	hook.Reset()

}

func TestBlockRequestGetCanonicalError(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	mdb := &mockDB{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, mdb)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Data: request1,
		Peer: p2p.Peer{},
	}
	mdb.getError = true

	ss.blockRequestBySlot <- msg1
	AssertLogsContain(t, hook, "Error retrieving block from db mock get canonical block error")
	hook.Reset()

}

func TestBlockRequestBySlot(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	mdb := &mockDB{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, mdb)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- msg1
	ss.cancel()
	<-exitRoutine
	AssertLogsContain(t, hook, "Sending requested block to peer")
	hook.Reset()
}

func TestStartEmptyState(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := DefaultConfig()
	ms := &mockChainService{}
	mdb := &mockDB{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms, mdb)

	ss.Start()
	AssertLogsContain(t, hook, "Empty chain state, but continue sync")

	hook.Reset()
	mdb.hasState = true

	ss.Start()
	AssertLogsDoNotContain(t, hook, "Empty chain state, but continue sync")

	ss.cancel()
}
