package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p-core"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func init() {
	allowedBlocksPerSecond = 64
	allowedBlocksBurst = int64(10 * allowedBlocksPerSecond)
}

func TestService_StatusZeroEpoch(t *testing.T) {
	bState, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 0})
	if err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p:         p2ptest.NewTestP2P(t),
		initialSync: new(mockSync.Sync),
		chain: &mockChain.ChainService{
			Genesis: time.Now(),
			State:   bState,
		},
	}
	r.chainStarted = true

	err = r.Status()
	if err != nil {
		t.Errorf("Wanted non failing status but got: %v", err)
	}
}

func TestService_Stop_SendGoodbyeToAllPeers(t *testing.T) {
	bState, err := stateTrie.InitializeFromProto(&pb.BeaconState{Slot: 0})
	if err != nil {
		t.Fatal(err)
	}
	p1 := p2ptest.NewTestP2P(t)
	r1 := &Service{
		p2p:         p1,
		initialSync: new(mockSync.Sync),
		chain: &mockChain.ChainService{
			Genesis: time.Now(),
			State:   bState,
		},
	}

	p2 := p2ptest.NewTestP2P(t)
	r2 := &Service{
		p2p:         p2,
		initialSync: new(mockSync.Sync),
		chain: &mockChain.ChainService{
			Genesis: time.Now(),
		},
	}
	p1.Connect(p2)

	var wg sync.WaitGroup
	wg.Add(1)
	handler := func(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
		wg.Done()
		return nil
	}
	r2.registerRPC(
		p2p.RPCGoodByeTopic,
		new(uint64),
		handler,
	)

	err = r1.Stop()
	if err != nil {
		t.Errorf("Error stopping service: %v", err)
	}

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Errorf("Did not receive RPC in 1 second")
	}
}
