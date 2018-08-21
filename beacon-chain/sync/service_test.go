package sync

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
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

func (mp *mockP2P) Subscribe(msg interface{}, channel interface{}) event.Subscription {
	return new(event.Feed).Subscribe(channel)
}

func (mp *mockP2P) Broadcast(msg interface{}) {}

func (mp *mockP2P) Send(msg interface{}, peer p2p.Peer) {
}

type mockChainService struct {
	processedBlockHashes [][32]byte
}

func (ms *mockChainService) ProcessBlock(b *types.Block) {
	h, _ := b.Hash()
	if ms.processedBlockHashes == nil {
		ms.processedBlockHashes = [][32]byte{}
	}
	ms.processedBlockHashes = append(ms.processedBlockHashes, h)
}

func (ms *mockChainService) ContainsBlock(h [32]byte) bool {
	for _, h1 := range ms.processedBlockHashes {
		if h == h1 {
			return true
		}
	}
	return false
}

func (ms *mockChainService) ProcessedBlockHashes() [][32]byte {
	return ms.processedBlockHashes
}

func (ms *mockChainService) SaveBlock(block *types.Block) error {
	return nil
}

func (ms *mockChainService) HasStoredState() (bool, error) {
	return false, nil
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, &mockChainService{})

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
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
		ss.run(ss.ctx.Done())
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

	block := types.NewBlock(data)
	h, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if ms.processedBlockHashes[0] != h {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h, ms.processedBlockHashes[0])
	}
	hook.Reset()
}

func TestProcessMultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
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

	block1 := types.NewBlock(data1)
	h1, err := block1.Hash()
	if err != nil {
		t.Fatal(err)
	}

	block2 := types.NewBlock(data2)
	h2, err := block2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Sync service broadcasts the two separate blocks
	// and forwards them to to the local chain.
	if ms.processedBlockHashes[0] != h1 {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h1, ms.processedBlockHashes[0])
	}
	if ms.processedBlockHashes[1] != h2 {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h2, ms.processedBlockHashes[1])
	}
	hook.Reset()
}

func TestProcessSameBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockHashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
		exitRoutine <- true
	}()

	data := &pb.BeaconBlock{
		PowChainRef: []byte{1, 2, 3},
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
	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	block := types.NewBlock(data)
	h, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Sync service broadcasts the two separate blocks
	// and forwards them to to the local chain.
	if len(ms.processedBlockHashes) > 1 {
		t.Error("Should have only processed one block, processed both instead")
	}
	if ms.processedBlockHashes[0] != h {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h, ms.processedBlockHashes[0])
	}
	hook.Reset()
}

func TestSetBlockForInitialSync(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockBufferSize: 0, CrystallizedStateBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	go func() {
		ss.runInitialSync(delayChan, ss.ctx.Done())
		exitRoutine <- true
	}()

	generichash := make([]byte, 32)
	generichash[0] = 'a'

	block := &pb.BeaconBlock{
		PowChainRef:           []byte{1, 2, 3},
		ParentHash:            generichash,
		SlotNumber:            uint64(20),
		CrystallizedStateHash: generichash,
	}

	blockResponse := &pb.BeaconBlockResponse{Block: block}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	var hash [32]byte
	copy(hash[:], blockResponse.Block.CrystallizedStateHash)

	if hash != ss.initialCrystallizedStateHash {
		t.Fatalf("Crystallized state hash not updated: %x", blockResponse.Block.CrystallizedStateHash)
	}

	hook.Reset()

}

func TestSavingBlocksInSync(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{BlockBufferSize: 0, CrystallizedStateBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	go func() {
		ss.runInitialSync(delayChan, ss.ctx.Done())
		exitRoutine <- true
	}()

	generichash := make([]byte, 32)
	generichash[0] = 'a'

	crystallizedState := &pb.CrystallizedState{
		LastFinalizedSlot: 99,
	}

	stateResponse := &pb.CrystallizedStateResponse{
		CrystallizedState: crystallizedState,
	}

	incorrectState := &pb.CrystallizedState{
		LastFinalizedSlot: 9,
		LastJustifiedSlot: 20,
	}

	incorrectStateResponse := &pb.CrystallizedStateResponse{
		CrystallizedState: incorrectState,
	}

	crystallizedStateHash, err := types.NewCrystallizedState(crystallizedState).Hash()
	if err != nil {
		t.Fatalf("unable to get hash of crystallized state: %v", err)
	}

	getBlockResponseMsg := func(slotNumber uint64) p2p.Message {
		block := &pb.BeaconBlock{
			PowChainRef:           []byte{1, 2, 3},
			ParentHash:            generichash,
			SlotNumber:            slotNumber,
			CrystallizedStateHash: crystallizedStateHash[:],
		}

		blockResponse := &pb.BeaconBlockResponse{
			Block: block,
		}

		return p2p.Message{
			Peer: p2p.Peer{},
			Data: blockResponse,
		}
	}

	msg1 := getBlockResponseMsg(0)

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: incorrectStateResponse,
	}

	ss.blockBuf <- msg1
	ss.crystallizedStateBuf <- msg2

	if ss.currentSlotNumber == incorrectStateResponse.CrystallizedState.LastFinalizedSlot {
		t.Fatalf("Crystallized state updated incorrectly: %x", ss.currentSlotNumber)
	}

	msg2.Data = stateResponse

	ss.crystallizedStateBuf <- msg2

	if crystallizedStateHash != ss.initialCrystallizedStateHash {
		br := msg1.Data.(*pb.BeaconBlockResponse)
		t.Fatalf("Crystallized state hash not updated: %x", br.Block.CrystallizedStateHash)
	}

	msg1 = getBlockResponseMsg(30)
	ss.blockBuf <- msg1

	if stateResponse.CrystallizedState.GetLastFinalizedSlot() != ss.currentSlotNumber {
		t.Fatalf("slotnumber saved when it was not supposed too: %v", stateResponse.CrystallizedState.GetLastFinalizedSlot())
	}

	msg1 = getBlockResponseMsg(100)
	ss.blockBuf <- msg1

	ss.cancel()
	<-exitRoutine

	br := msg1.Data.(*pb.BeaconBlockResponse)
	if br.Block.GetSlotNumber() != ss.currentSlotNumber {
		t.Fatalf("slotnumber not updated despite receiving a valid block: %v", ss.currentSlotNumber)
	}

	hook.Reset()

}

func TestDelayChan(t *testing.T) {
	hook := logTest.NewGlobal()
	cfg := Config{BlockBufferSize: 0, CrystallizedStateBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)
	delayChan := make(chan time.Time)

	go func() {
		ss.runInitialSync(delayChan, ss.ctx.Done())
		exitRoutine <- true
	}()

	generichash := make([]byte, 32)
	generichash[0] = 'a'

	crystallizedstate := &pb.CrystallizedState{
		LastFinalizedSlot: 99,
	}

	stateResponse := &pb.CrystallizedStateResponse{
		CrystallizedState: crystallizedstate,
	}

	crystallizedStateHash, err := types.NewCrystallizedState(stateResponse.CrystallizedState).Hash()
	if err != nil {
		t.Fatalf("unable to get hash of crystallized state: %v", err)
	}

	block := &pb.BeaconBlock{
		PowChainRef:           []byte{1, 2, 3},
		ParentHash:            generichash,
		SlotNumber:            uint64(20),
		CrystallizedStateHash: crystallizedStateHash[:],
	}

	blockResponse := &pb.BeaconBlockResponse{
		Block: block,
	}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: stateResponse,
	}

	ss.blockBuf <- msg1

	ss.crystallizedStateBuf <- msg2

	blockResponse.Block.SlotNumber = 100
	msg1.Data = blockResponse

	ss.blockBuf <- msg1

	delayChan <- time.Time{}

	ss.cancel()
	<-exitRoutine

	testutil.AssertLogsContain(t, hook, "Exiting initial sync and starting normal sync")

	hook.Reset()

}
