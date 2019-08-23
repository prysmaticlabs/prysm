package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestHelloRPCHandler_Disconnects_OnForkVersionMismatch(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

	r := &RegularSync{p2p: p1}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		code, errMsg, err := r.readStatusCode(stream)
		if err != nil {
			t.Fatal(err)
		}
		if code == 0 {
			t.Error("Expected a non-zero code")
		}
		if errMsg.ErrorMessage != errWrongForkVersion.Error() {
			t.Errorf("Received unexpected message response in the stream: %+v", err)
		}
	})

	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.helloRPCHandler(context.Background(), &pb.Hello{ForkVersion: []byte("fake")}, stream1)
	if err != errWrongForkVersion {
		t.Errorf("Expected error %v, got %v", errWrongForkVersion, err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.Host.Network().Peers()) != 0 {
		t.Error("handler did not disconnect peer")
	}
}

func TestHelloRPCHandler_ReturnsHelloMessage(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

	// Set up a head state with data we expect.
	headRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 111})
	if err != nil {
		t.Fatal(err)
	}
	finalizedRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 40})
	if err != nil {
		t.Fatal(err)
	}
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	genesisState.Slot = 111
	genesisState.BlockRoots[111%params.BeaconConfig().SlotsPerHistoricalRoot] = headRoot[:]
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  finalizedRoot[:],
	}

	r := &RegularSync{
		p2p: p1,
		chain: &mockChainService{
			headState:        genesisState,
			finalizedCheckpt: finalizedCheckpt,
			headRoot:         headRoot[:],
		},
	}

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Hello{}
		if err := r.p2p.Encoding().Decode(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Hello{
			ForkVersion:    params.BeaconConfig().GenesisForkVersion,
			HeadSlot:       genesisState.Slot,
			HeadRoot:       headRoot[:],
			FinalizedEpoch: 5,
			FinalizedRoot:  finalizedRoot[:],
		}
		if !proto.Equal(out, expected) {
			t.Errorf("Did not receive expected message. Got %+v wanted %+v", out, expected)
		}
	})
	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.helloRPCHandler(context.Background(), &pb.Hello{ForkVersion: params.BeaconConfig().GenesisForkVersion}, stream1)
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestRegularSync_HelloRPCHandler_AddsHandshake(t *testing.T) {
	t.Skip("TODO(3147): Add a test to ensure the handshake was added.")
}
