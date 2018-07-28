package sync

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/event"
	blake2b "github.com/minio/blake2b-simd"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type mockP2P struct{}

func (mp *mockP2P) Feed(msg interface{}) *event.Feed {
	return new(event.Feed)
}

func (mp *mockP2P) Broadcast(msg interface{}) {}

type mockChainService struct {
	processedHashes [][32]byte
}

func (ms *mockChainService) ProcessBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	if ms.processedHashes == nil {
		ms.processedHashes = [][32]byte{}
	}
	ms.processedHashes = append(ms.processedHashes, h)
	return nil
}

func (ms *mockChainService) ContainsBlock(h [32]byte) bool {
	for _, h1 := range ms.processedHashes {
		if h == h1 {
			return true
		}
	}
	return false
}

func (ms *mockChainService) ProcessedHashes() [][32]byte {
	return ms.processedHashes
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, &mockChainService{})

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
		exitRoutine <- true
	}()

	announceHash := blake2b.Sum256([]byte{})
	hashAnnounce := pb.BeaconBlockHashAnnounce{
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

	testutil.AssertLogsContain(t, hook, "Requesting full block data from sender")
	hook.Reset()
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
		exitRoutine <- true
	}()

	blockResponse := pb.BeaconBlockResponse{
		MainChainRef: []byte{1, 2, 3, 4, 5},
	}

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}

	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	block, err := types.NewBlockWithData(&blockResponse)
	if err != nil {
		t.Fatalf("Could not instantiate new block from proto: %v", err)
	}
	h, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Sync service broadcasts the block and forwards the block to to the local chain.
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h))

	if ms.processedHashes[0] != h {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h, ms.processedHashes[0])
	}
	hook.Reset()
}

func TestProcessMultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
		exitRoutine <- true
	}()

	blockResponse1 := pb.BeaconBlockResponse{
		MainChainRef: []byte{1, 2, 3, 4, 5},
	}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse1,
	}

	blockResponse2 := pb.BeaconBlockResponse{
		MainChainRef: []byte{6, 7, 8, 9, 10},
	}

	msg2 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse2,
	}

	ss.blockBuf <- msg1
	ss.blockBuf <- msg2
	ss.cancel()
	<-exitRoutine

	block1, err := types.NewBlockWithData(&blockResponse1)
	if err != nil {
		t.Fatalf("Could not instantiate new block from proto: %v", err)
	}
	h1, err := block1.Hash()
	if err != nil {
		t.Fatal(err)
	}

	block2, err := types.NewBlockWithData(&blockResponse2)
	if err != nil {
		t.Fatalf("Could not instantiate new block from proto: %v", err)
	}
	h2, err := block2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Sync service broadcasts the two separate blocks
	// and forwards them to to the local chain.
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h1))
	if ms.processedHashes[0] != h1 {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h1, ms.processedHashes[0])
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h2))
	if ms.processedHashes[1] != h2 {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h2, ms.processedHashes[1])
	}
	hook.Reset()
}

func TestProcessSameBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ms := &mockChainService{}
	ss := NewSyncService(context.Background(), cfg, &mockP2P{}, ms)

	exitRoutine := make(chan bool)

	go func() {
		ss.run(ss.ctx.Done())
		exitRoutine <- true
	}()

	blockResponse := pb.BeaconBlockResponse{
		MainChainRef: []byte{1, 2, 3},
	}

	msg := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse,
	}
	ss.blockBuf <- msg
	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	block, err := types.NewBlockWithData(&blockResponse)
	if err != nil {
		t.Fatalf("Could not instantiate new block from proto: %v", err)
	}
	h, err := block.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// Sync service broadcasts the two separate blocks
	// and forwards them to to the local chain.
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h))
	if len(ms.processedHashes) > 1 {
		t.Error("should have only processed one block, processed both instead")
	}
	if ms.processedHashes[0] != h {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h, ms.processedHashes[0])
	}
	hook.Reset()
}
