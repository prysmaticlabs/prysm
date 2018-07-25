package sync

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
	"golang.org/x/crypto/blake2b"
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
	ss := NewSyncService(context.Background(), cfg)

	ns := MockNetworkService{}
	cs := MockChainService{}
	ss.SetNetworkService(&ns)
	ss.SetChainService(&cs)

	exitRoutine := make(chan bool)

	go func() {
		run(ss.ctx.Done(), ss.hashBuf, ss.blockBuf, &ns, &cs)
		exitRoutine <- true
	}()

	h, err := blake2b.New256(nil)
	if err != nil {
		t.Errorf("failed to intialize hash: %v", err)
	}

	// if a new hash is processed
	ss.ReceiveBlockHash(h)

	ss.cancel()
	<-exitRoutine

	// sync service requests the contents of the block and broadcasts the hash to peers
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("requesting block: %x", h.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("broadcasting hash: %x", h.Sum(nil)))
}

func TestProcessBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg)

	ns := MockNetworkService{}
	cs := MockChainService{}

	ss.SetNetworkService(&ns)
	ss.SetChainService(&cs)

	exitRoutine := make(chan bool)

	go func() {
		run(ss.ctx.Done(), ss.hashBuf, ss.blockBuf, &ns, &cs)
		exitRoutine <- true
	}()

	b := types.NewBlock(0)
	h, err := b.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// if the hash and the block are processed in order
	ss.ReceiveBlockHash(h)
	ss.ReceiveBlock(b)
	ss.cancel()
	<-exitRoutine

	// sync service broadcasts the block and forwards the block to to the local chain
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("broadcasting block: %x", h.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("forwarding block: %x", h.Sum(nil)))
}

func TestProcessMultipleBlocks(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg)

	ns := MockNetworkService{}
	cs := MockChainService{}

	ss.SetNetworkService(&ns)
	ss.SetChainService(&cs)

	exitRoutine := make(chan bool)

	go func() {
		run(ss.ctx.Done(), ss.hashBuf, ss.blockBuf, &ns, &cs)
		exitRoutine <- true
	}()

	b1 := types.NewBlock(0)
	h1, err := b1.Hash()
	if err != nil {
		t.Fatal(err)
	}

	b2 := types.NewBlock(1)
	h2, err := b2.Hash()
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(h1.Sum(nil), h2.Sum(nil)) {
		t.Fatalf("two blocks should not have the same hash:\n%x\n%x", h1.Sum(nil), h2.Sum(nil))
	}

	// if two different blocks are submitted
	ss.ReceiveBlockHash(h1)
	ss.ReceiveBlock(b1)
	ss.ReceiveBlockHash(h2)
	ss.ReceiveBlock(b2)
	ss.cancel()
	<-exitRoutine

	// both blocks are processed
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("broadcasting block: %x", h1.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("forwarding block: %x", h1.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("broadcasting block: %x", h2.Sum(nil)))
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("forwarding block: %x", h2.Sum(nil)))
}

func TestProcessSameBlock(t *testing.T) {
	hook := logTest.NewGlobal()

	cfg := Config{HashBufferSize: 0, BlockBufferSize: 0}
	ss := NewSyncService(context.Background(), cfg)

	ns := MockNetworkService{}
	cs := MockChainService{}

	ss.SetNetworkService(&ns)
	ss.SetChainService(&cs)

	exitRoutine := make(chan bool)

	go func() {
		run(ss.ctx.Done(), ss.hashBuf, ss.blockBuf, &ns, &cs)
		exitRoutine <- true
	}()

	b := types.NewBlock(0)
	h, err := b.Hash()
	if err != nil {
		t.Fatal(err)
	}

	// if the same block is processed twice
	ss.ReceiveBlockHash(h)
	ss.ReceiveBlock(b)
	ss.ReceiveBlockHash(h)
	// there's a tricky race condition where the second hash can sneak into the goroutine
	// before the first block inserts itself into the chain. therefore, its important
	// for hook.Reset() to be called after the second ProcessBlockHash call
	hook.Reset()
	ss.ReceiveBlock(b)
	ss.cancel()
	<-exitRoutine

	// the block isn't processed the second time
	testutil.AssertLogsDoNotContain(t, hook, fmt.Sprintf("broadcasting block: %x", h.Sum(nil)))
	testutil.AssertLogsDoNotContain(t, hook, fmt.Sprintf("forwarding block: %x", h.Sum(nil)))
}
