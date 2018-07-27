package sync

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"testing"

	"github.com/ethereum/go-ethereum/event"
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
	processedHashes []hash.Hash
}

func (ms *mockChainService) ProcessBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	if ms.processedHashes == nil {
		ms.processedHashes = []hash.Hash{}
	}
	ms.processedHashes = append(ms.processedHashes, h)
	return nil
}

func (ms *mockChainService) ContainsBlock(h hash.Hash) bool {
	for _, h1 := range ms.processedHashes {
		if bytes.Equal(h.Sum(nil), h1.Sum(nil)) {
			return true
		}
	}
	return false
}

func (ms *mockChainService) ProcessedHashes() []hash.Hash {
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

	hashAnnounce := pb.BeaconBlockHashAnnounce{
		Hash: []byte("hi"),
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
		MainChainRef: []byte("hi"),
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
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h.Sum(nil)))

	if !bytes.Equal(ms.processedHashes[0].Sum(nil), h.Sum(nil)) {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h.Sum(nil), ms.processedHashes[0].Sum(nil))
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
		MainChainRef: []byte("foo"),
	}

	msg1 := p2p.Message{
		Peer: p2p.Peer{},
		Data: blockResponse1,
	}

	blockResponse2 := pb.BeaconBlockResponse{
		MainChainRef: []byte("bar"),
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
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h1.Sum(nil)))
	if !bytes.Equal(ms.processedHashes[0].Sum(nil), h1.Sum(nil)) {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h1.Sum(nil), ms.processedHashes[0].Sum(nil))
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h2.Sum(nil)))
	if !bytes.Equal(ms.processedHashes[1].Sum(nil), h2.Sum(nil)) {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h2.Sum(nil), ms.processedHashes[1].Sum(nil))
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
		MainChainRef: []byte("foo"),
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
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting block hash to peers: %x", h.Sum(nil)))
	if len(ms.processedHashes) > 1 {
		t.Error("should have only processed one block, processed both instead")
	}
	if !bytes.Equal(ms.processedHashes[0].Sum(nil), h.Sum(nil)) {
		t.Errorf("Expected processed hash to be equal to block hash. wanted=%x, got=%x", h.Sum(nil), ms.processedHashes[0].Sum(nil))
	}
	hook.Reset()
}
