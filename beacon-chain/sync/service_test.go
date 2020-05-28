package sync

import (
	"context"
	"testing"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mockChain "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

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

func TestSyncHandlers_WaitToSync(t *testing.T) {
	p2p := p2ptest.NewTestP2P(t)
	chainService := &mockChain.ChainService{
		Genesis:        time.Now(),
		ValidatorsRoot: [32]byte{'A'},
	}
	r := Service{
		ctx:           context.Background(),
		p2p:           p2p,
		chain:         chainService,
		stateNotifier: chainService.StateNotifier(),
		initialSync:   &mockSync.Sync{IsSyncing: false},
	}

	topic := "/eth2/%x/beacon_block"
	go r.registerHandlers()
	time.Sleep(100 * time.Millisecond)
	i := r.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.Initialized,
		Data: &statefeed.InitializedData{
			StartTime: time.Now(),
		},
	})
	if i == 0 {
		t.Fatal("didn't send genesis time to subscribers")
	}
	b := []byte("sk")
	b32 := bytesutil.ToBytes32(b)
	sk, err := bls.SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}

	msg := &ethpb.SignedBeaconBlock{
		Block: &ethpb.BeaconBlock{
			ParentRoot: testutil.Random32Bytes(t),
		},
		Signature: sk.Sign([]byte("data")).Marshal(),
	}
	p2p.ReceivePubSub(topic, msg)
	// wait for chainstart to be sent
	time.Sleep(400 * time.Millisecond)
	if !r.chainStarted {
		t.Fatal("Did not receive chain start event.")
	}

}
