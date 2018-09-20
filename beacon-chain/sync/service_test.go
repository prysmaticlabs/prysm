package sync

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
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

type mockChainService struct {
	slotExists bool
	checkError bool
	getError   bool
}

func (ms *mockChainService) ContainsBlock(h [32]byte) (bool, error) {
	return false, nil
}

func (ms *mockChainService) HasStoredState() (bool, error) {
	return false, nil
}

func (ms *mockChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) IncomingAttestationFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockChainService) CurrentCrystallizedState() *types.CrystallizedState {
	cState, err := types.NewGenesisCrystallizedState()
	if err != nil {
		fmt.Println(err)
	}
	return cState
}

func (ms *mockChainService) BlockSlotNumberByHash(h [32]byte) (uint64, error) {
	return 0, nil
}

func (ms *mockChainService) CheckForCanonicalBlockBySlot(slotnumber uint64) (bool, error) {
	if ms.checkError {
		return ms.slotExists, errors.New("mock check canonical block error")
	}
	return ms.slotExists, nil
}

func (ms *mockChainService) CanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error) {
	if ms.getError {
		return nil, errors.New("mock get canonical block error")
	}
	if !ms.slotExists {
		return nil, errors.New("invalid key")
	}
	return types.NewBlock(&pb.BeaconBlock{SlotNumber: slotnumber}), nil
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, &mockChainService{})

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

	testutil.AssertLogsContain(t, hook, "requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run()
		exitRoutine <- true
	}()

	data := &pb.BeaconBlock{
		PowChainRef: []byte{1, 2, 3, 4, 5},
		ParentHash:  make([]byte, 32),
	}
	attestation := &pb.AggregatedAttestation{
		Slot:           0,
		ShardId:        0,
		ShardBlockHash: []byte{'A'},
	}

	responseBlock := &pb.BeaconBlockResponse{
		Block:       data,
		Attestation: attestation,
	}

	msg := p2p.Message{
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

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

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
	testutil.AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	testutil.AssertLogsContain(t, hook, "Sending newly received block to subscribers")
	hook.Reset()
}

func TestBlockRequestErrors(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

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
	testutil.AssertLogsContain(t, hook, "Received malformed beacon block request p2p message")

	request1 := &pb.BeaconBlockRequestBySlotNumber{
		SlotNumber: 20,
	}

	msg1 := p2p.Message{
		Data: request1,
		Peer: p2p.Peer{},
	}

	ss.blockRequestBySlot <- msg1
	testutil.AssertLogsDoNotContain(t, hook, "Sending requested block to peer")
	hook.Reset()

}

func TestBlockRequestGetCanonicalError(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

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
	ms.slotExists = true
	ms.getError = true

	ss.blockRequestBySlot <- msg1
	testutil.AssertLogsContain(t, hook, "Error retrieving block from db mock get canonical block error")
	hook.Reset()

}

func TestBlockRequestBySlot(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0, BlockRequestBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

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

	ms.checkError = true
	ms.slotExists = true

	ss.blockRequestBySlot <- msg1
	testutil.AssertLogsContain(t, hook, "Error checking db for block mock check canonical block error")

	ms.checkError = false

	ss.blockRequestBySlot <- msg1
	ss.cancel()
	<-exitRoutine
	testutil.AssertLogsContain(t, hook, "Sending requested block to peer")
	hook.Reset()
}

type mockEmptyChainService struct {
	hasStoredState bool
}

func (ms *mockEmptyChainService) ContainsBlock(h [32]byte) (bool, error) {
	return false, nil
}

func (ms *mockEmptyChainService) HasStoredState() (bool, error) {
	return ms.hasStoredState, nil
}

func (ms *mockEmptyChainService) IncomingBlockFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockEmptyChainService) IncomingAttestationFeed() *event.Feed {
	return new(event.Feed)
}

func (ms *mockEmptyChainService) setState(flag bool) {
	ms.hasStoredState = flag
}

func (ms *mockEmptyChainService) CheckForCanonicalBlockBySlot(slotnumber uint64) (bool, error) {
	return false, nil
}

func (ms *mockEmptyChainService) CanonicalBlockBySlotNumber(slotnumber uint64) (*types.Block, error) {
	return nil, nil
}

func (ms *mockEmptyChainService) CurrentCrystallizedState() *types.CrystallizedState {
	return types.NewCrystallizedState(nil)
}

func (ms *mockEmptyChainService) BlockSlotNumberByHash(h [32]byte) (uint64, error) {
	return 0, nil
}

func TestStartEmptyState(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := DefaultConfig()
	ms := &mockEmptyChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	ss.Start()
	testutil.AssertLogsContain(t, hook, "Empty chain state, but continue sync")

	hook.Reset()
	ms.setState(true)

	ss.Start()
	testutil.AssertLogsDoNotContain(t, hook, "Empty chain state, but continue sync")

	ss.cancel()
}
