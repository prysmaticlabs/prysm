package sync

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/v1"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/blake2s"
)

var testLog = log.WithField("prefix", "sync_test")

// MockChainService implements a simplified local chain that stores blocks in a slice
type MockChainService struct {
	processedHashes []hash.Hash
}

func (ms *MockChainService) ProcessBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	if ms.processedHashes == nil {
		ms.processedHashes = []hash.Hash{}
	}
	log.Info("Processed block with hash: %x", h)
	ms.processedHashes = append(ms.processedHashes, h)
	return nil
}

func (ms *MockChainService) ContainsBlock(h hash.Hash) bool {
	for _, h1 := range ms.processedHashes {
		if bytes.Equal(h.Sum(nil), h1.Sum(nil)) {
			return true
		}
	}
	return false
}

func TestProcessBlockHash(t *testing.T) {
	hook := logTest.NewGlobal()

	// set the channel's buffer to 0 to make channel interactions blocking
	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	cs := &MockChainService{}
	beaconp2p, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("unable to setup beaconp2p: %v", err)
	}
	ss := NewSyncService(context.Background(), cfg, beaconp2p, cs)

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

	h, err := blake2s.New256([]byte("hi"))
	if err != nil {
		t.Errorf("failed to intialize hash: %v", err)
	}

	// Sync service requests the contents of the block and broadcasts the hash to peers.
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("Broadcasting blockhash to peers: %x", h.Sum(nil)))
	testutil.AssertLogsContain(t, hook, "Requesting full block data from sender")
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	cs := &MockChainService{}
	beaconp2p, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("unable to setup beaconp2p: %v", err)
	}
	ss := NewSyncService(context.Background(), cfg, beaconp2p, cs)

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

	// if a new hash is processed
	ss.blockBuf <- msg
	ss.cancel()
	<-exitRoutine

	// Sync service broadcasts the block and forwards the block to to the local chain.
	testutil.AssertLogsContain(t, hook, "Broadcasting block to peers")
	testutil.AssertLogsContain(t, hook, "Processed block")
}

// func TestProcessMultipleBlocks(t *testing.T) {
// }

// func TestProcessSameBlock(t *testing.T) {
// The block isn't processed the second time.
// }
