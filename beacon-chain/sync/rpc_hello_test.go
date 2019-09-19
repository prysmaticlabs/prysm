package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestHelloRPCHandler_Disconnects_OnForkVersionMismatch(t *testing.T) {
	// TODO(3441): Fix ssz string length issue.
	t.Skip("3441: SSZ is decoding a string with an unexpected length")
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

	r := &RegularSync{p2p: p1, helloTracker: make(map[peer.ID]*pb.Hello)}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		code, errMsg, err := ReadStatusCode(stream, p1.Encoding())
		if err != nil {
			t.Fatal(err)
		}
		if code == 0 {
			t.Error("Expected a non-zero code")
		}
		if errMsg != errWrongForkVersion.Error() {
			t.Logf("Received error string len %d, wanted error string len %d", len(errMsg), len(errWrongForkVersion.Error()))
			t.Errorf("Received unexpected message response in the stream: %s. Wanted %s.", errMsg, errWrongForkVersion.Error())
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
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
		},
		helloTracker: make(map[peer.ID]*pb.Hello),
	}

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Hello{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
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

func TestHandshakeHandlers_Roundtrip(t *testing.T) {
	// Scenario is that p1 and p2 connect, exchange handshakes.
	// p2 disconnects and p1 should forget the handshake status.
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)

	r := &RegularSync{
		p2p: p1,
		chain: &mock.ChainService{
			State: &pb.BeaconState{Slot: 5},
			FinalizedCheckPoint: &ethpb.Checkpoint{},
		},
		helloTracker: make(map[peer.ID]*pb.Hello),
		ctx:          context.Background(),
	}

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/hello/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Hello{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}

		resp := &pb.Hello{HeadSlot: 100, ForkVersion: params.BeaconConfig().GenesisForkVersion}

		if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
			t.Fatal(err)
		}
		_, err := r.p2p.Encoding().EncodeWithLength(stream, resp)
		if err != nil {
			t.Fatal(err)
		}
		stream.Close()
	})

	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	// Wait for stream buffer to be read.
	time.Sleep(200 * time.Millisecond)

	if len(r.helloTracker) != 1 {
		t.Errorf("Expected 1 status in the tracker, got %d", len(r.helloTracker))
	}

	if err := p2.Disconnect(p1.PeerID()); err != nil {
		t.Fatal(err)
	}

	// Wait for disconnect event to trigger.
	time.Sleep(200 * time.Millisecond)

	if len(r.helloTracker) != 0 {
		t.Errorf("Expected 0 status in the tracker, got %d", len(r.helloTracker))
	}

}

func TestHelloRPCRequest_RequestSent(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)

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
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
		},
		helloTracker: make(map[peer.ID]*pb.Hello),
		ctx:          context.Background(),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/hello/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Hello{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
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

	p1.AddConnectionHandler(r.sendRPCHelloRequest)
	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to continue being connected")
	}
}
