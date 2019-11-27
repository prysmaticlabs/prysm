package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestHelloRPCHandler_Disconnects_OnForkVersionMismatch(t *testing.T) {
	peerstatus.Clear()

	// TODO(3441): Fix ssz string length issue.
	t.Skip("3441: SSZ is decoding a string with an unexpected length")
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

	err = r.statusRPCHandler(context.Background(), &pb.Status{HeadForkVersion: []byte("fake")}, stream1)
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
	peerstatus.Clear()

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
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
		},
	}

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
			HeadSlot:        genesisState.Slot,
			HeadRoot:        headRoot[:],
			FinalizedEpoch:  5,
			FinalizedRoot:   finalizedRoot[:],
		}
		if !proto.Equal(out, expected) {
			t.Errorf("Did not receive expected message. Got %+v wanted %+v", out, expected)
		}
	})
	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.statusRPCHandler(context.Background(), &pb.Status{HeadForkVersion: params.BeaconConfig().GenesisForkVersion}, stream1)
	if err != nil {
		t.Errorf("Unxpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}

func TestHandshakeHandlers_Roundtrip(t *testing.T) {
	peerstatus.Clear()

	// Scenario is that p1 and p2 connect, exchange handshakes.
	// p2 disconnects and p1 should forget the handshake status.
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)

	r := &RegularSync{
		p2p: p1,
		chain: &mock.ChainService{
			State:               &pb.BeaconState{Slot: 5},
			FinalizedCheckPoint: &ethpb.Checkpoint{},
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
		},
		ctx: context.Background(),
	}

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}

		resp := &pb.Status{HeadSlot: 100, HeadForkVersion: params.BeaconConfig().GenesisForkVersion}

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

	if peerstatus.Count() != 1 {
		t.Errorf("Expected 1 status in the cache, got %d", peerstatus.Count())
	}

	if err := p2.Disconnect(p1.PeerID()); err != nil {
		t.Fatal(err)
	}

	// Wait for disconnect event to trigger.
	time.Sleep(200 * time.Millisecond)

	if peerstatus.Count() != 0 {
		t.Errorf("Expected 0 status in the tracker, got %d", peerstatus.Count())
	}

}

func TestStatusRPCRequest_RequestSent(t *testing.T) {
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
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
		},
		ctx: context.Background(),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			HeadForkVersion: params.BeaconConfig().GenesisForkVersion,
			HeadSlot:        genesisState.Slot,
			HeadRoot:        headRoot[:],
			FinalizedEpoch:  5,
			FinalizedRoot:   finalizedRoot[:],
		}
		if !proto.Equal(out, expected) {
			t.Errorf("Did not receive expected message. Got %+v wanted %+v", out, expected)
		}
	})

	p1.AddConnectionHandler(r.sendRPCStatusRequest)
	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to continue being connected")
	}
}

func TestStatusRPCRequest_BadPeerHandshake(t *testing.T) {
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
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
		},
		ctx: context.Background(),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithLength(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			HeadForkVersion: []byte{1, 1, 1, 1},
			HeadSlot:        genesisState.Slot,
			HeadRoot:        headRoot[:],
			FinalizedEpoch:  5,
			FinalizedRoot:   finalizedRoot[:],
		}
		if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
			log.WithError(err).Error("Failed to write to stream")
		}
		_, err := r.p2p.Encoding().EncodeWithLength(stream, expected)
		if err != nil {
			t.Errorf("Could not send status: %v", err)
		}
	})

	p1.AddConnectionHandler(r.sendRPCStatusRequest)
	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	time.Sleep(100 * time.Millisecond)

	if len(p1.Host.Network().Peers()) != 0 {
		t.Error("Expected peer to be disconnected")
	}

	if peerstatus.FailureCount(p2.PeerID()) != 1 {
		t.Errorf("Failure count was not bumped to one, instead it is %d", peerstatus.FailureCount(p2.PeerID()))
	}
}
