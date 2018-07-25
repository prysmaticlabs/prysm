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

type MockNetworkService struct{}

func (ns *MockNetworkService) BroadcastBlockHash(h hash.Hash) error {
	testLog.Infof("broadcasting hash: %x", h.Sum(nil))
	return nil
}

func (ns *MockNetworkService) BroadcastBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	testLog.Infof("broadcasting block: %x", h.Sum(nil))
	return nil
}

func (ns *MockNetworkService) RequestBlock(h hash.Hash) error {
	testLog.Infof("requesting block: %x", h.Sum(nil))
	return nil
}

// MockChainService implements a simplified local chain that stores blocks in a slice
type MockChainService struct {
	processedHashes []hash.Hash
}

func (ms *MockChainService) ProcessBlock(b *types.Block) error {
	h, err := b.Hash()
	if err != nil {
		return err
	}

	testLog.Infof("forwarding block: %x", h.Sum(nil))
	if ms.processedHashes == nil {
		ms.processedHashes = []hash.Hash{}
	}

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
	ns := &MockNetworkService{}
	cs := &MockChainService{}
	beaconp2p, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("unable to setup beaconp2p: %v", err)
	}
	ss := NewSyncService(context.Background(), cfg, beaconp2p, ns, cs)

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
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("broadcasting hash: %x", h.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("requesting block: %x", h.Sum(nil)))
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ns := &MockNetworkService{}
	cs := &MockChainService{}
	beaconp2p, err := p2p.NewServer()
	if err != nil {
		t.Fatalf("unable to setup beaconp2p: %v", err)
	}
	ss := NewSyncService(context.Background(), cfg, beaconp2p, ns, cs)

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
	testutil.AssertLogsContain(t, hook, "broadcasting block")
	testutil.AssertLogsContain(t, hook, "forwarding block")
}

// func TestProcessMultipleBlocks(t *testing.T) {
// }

// func TestProcessSameBlock(t *testing.T) {
// The block isn't processed the second time.
// }
