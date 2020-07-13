package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testingDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStatusRPCHandler_Disconnects_OnForkVersionMismatch(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	root := [32]byte{'C'}

	r := &Service{p2p: p1,
		chain: &mock.ChainService{
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  root[:],
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		}}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.FinalizedRoot, root[:]) {
			t.Errorf("Expected finalized root of %#x but got %#x", root, out.FinalizedRoot)
		}
	})

	pcl2 := protocol.ID("/eth2/beacon_chain/req/goodbye/1/ssz_snappy")
	var wg2 sync.WaitGroup
	wg2.Add(1)
	p2.BHost.SetStreamHandler(pcl2, func(stream network.Stream) {
		defer wg2.Done()
		msg := new(uint64)
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, msg); err != nil {
			t.Error(err)
		}
		if *msg != codeWrongNetwork {
			t.Errorf("Wrong goodbye code: %d", *msg)
		}

	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.statusRPCHandler(context.Background(), &pb.Status{ForkDigest: []byte("fake")}, stream1)
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	if testutil.WaitTimeout(&wg2, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.BHost.Network().Peers()) != 0 {
		t.Error("handler did not disconnect peer")
	}
}

func TestStatusRPCHandler_ConnectsOnGenesis(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	root := [32]byte{}

	r := &Service{p2p: p1,
		chain: &mock.ChainService{
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
				Root:  params.BeaconConfig().ZeroHash[:],
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		}}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.FinalizedRoot, root[:]) {
			t.Errorf("Expected finalized root of %#x but got %#x", root, out.FinalizedRoot)
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	err = r.statusRPCHandler(context.Background(), &pb.Status{ForkDigest: digest[:], FinalizedRoot: params.BeaconConfig().ZeroHash[:]}, stream1)
	if err != nil {
		t.Errorf("Expected no error but got %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("handler disconnected with peer")
	}
}

func TestStatusRPCHandler_ReturnsHelloMessage(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	db, _ := testingDB.SetupDB(t)

	// Set up a head state with data we expect.
	headRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 111})
	if err != nil {
		t.Fatal(err)
	}
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: blkSlot})
	if err != nil {
		t.Fatal(err)
	}
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if err := genesisState.SetSlot(111); err != nil {
		t.Fatal(err)
	}
	if err := genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: blkSlot}}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(context.Background(), finalizedRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 3,
		Root:  finalizedRoot[:],
	}
	totalSec := params.BeaconConfig().SlotsPerEpoch * 5 * params.BeaconConfig().SecondsPerSlot
	genTime := time.Now().Unix() - int64(totalSec)

	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			ValidatorsRoot: [32]byte{'A'},
			Genesis:        time.Unix(genTime, 0),
		},
		db: db,
	}
	digest, err := r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	// Setup streams
	pcl := protocol.ID("/testing")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			ForkDigest:     digest[:],
			HeadSlot:       genesisState.Slot(),
			HeadRoot:       headRoot[:],
			FinalizedEpoch: 3,
			FinalizedRoot:  finalizedRoot[:],
		}
		if !proto.Equal(out, expected) {
			t.Errorf("Did not receive expected message. Got %+v wanted %+v", out, expected)
		}
	})
	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.statusRPCHandler(context.Background(), &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  finalizedRoot[:],
		FinalizedEpoch: 3,
	}, stream1)
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
	db, _ := testingDB.SetupDB(t)

	p1.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'A', 'B'},
	}

	p2.LocalMetadata = &pb.MetaData{
		SeqNumber: 2,
		Attnets:   []byte{'C', 'D'},
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	blk := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 0}}
	if err := db.SaveBlock(context.Background(), blk); err != nil {
		t.Fatal(err)
	}
	finalizedRoot, err := ssz.HashTreeRoot(blk.Block)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(context.Background(), finalizedRoot); err != nil {
		t.Fatal(err)
	}
	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               st,
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0, Root: finalizedRoot[:]},
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		db:  db,
		ctx: context.Background(),
	}
	p1.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	r2 := &Service{
		chain: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0, Root: finalizedRoot[:]},
		},
		p2p: p2,
	}
	p2.Digest, err = r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		log.WithField("status", out).Warn("received status")
		resp := &pb.Status{HeadSlot: 100, ForkDigest: p2.Digest[:],
			FinalizedRoot: finalizedRoot[:], FinalizedEpoch: 0}

		if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
			t.Fatal(err)
		}
		_, err := r.p2p.Encoding().EncodeWithMaxLength(stream, resp)
		if err != nil {
			t.Fatal(err)
		}
		log.WithField("status", out).Warn("sending status")
		if err := stream.Close(); err != nil {
			t.Log(err)
		}
	})

	pcl = "/eth2/beacon_chain/req/ping/1/ssz_snappy"
	var wg2 sync.WaitGroup
	wg2.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg2.Done()
		out := new(uint64)
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		if *out != 2 {
			t.Fatalf("Wanted 2 but got %d as our sequence number", *out)
		}
		err := r2.pingHandler(context.Background(), out, stream)
		if err != nil {
			t.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
	})

	numInactive1 := len(p1.Peers().Inactive())
	numActive1 := len(p1.Peers().Active())

	p1.Connect(p2)

	p1.Peers().Add(new(enr.Record), p2.BHost.ID(), p2.BHost.Addrs()[0], network.DirUnknown)
	p1.Peers().SetMetadata(p2.BHost.ID(), p2.LocalMetadata)

	p2.Peers().Add(new(enr.Record), p1.BHost.ID(), p1.BHost.Addrs()[0], network.DirUnknown)
	p2.Peers().SetMetadata(p1.BHost.ID(), p1.LocalMetadata)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	if testutil.WaitTimeout(&wg2, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	// Wait for stream buffer to be read.
	time.Sleep(200 * time.Millisecond)

	numInactive2 := len(p1.Peers().Inactive())
	numActive2 := len(p1.Peers().Active())

	if numInactive2 != numInactive1 {
		t.Errorf("Number of inactive peers changed unexpectedly: was %d, now %d", numInactive1, numInactive2)
	}
	if numActive2 != numActive1+1 {
		t.Errorf("Number of active peers unexpected: wanted %d, found %d", numActive1+1, numActive2)
	}

	if err := p2.Disconnect(p1.PeerID()); err != nil {
		t.Fatal(err)
	}
	p1.Peers().SetConnectionState(p2.PeerID(), peers.PeerDisconnected)

	// Wait for disconnect event to trigger.
	time.Sleep(200 * time.Millisecond)

	numInactive3 := len(p1.Peers().Inactive())
	numActive3 := len(p1.Peers().Active())
	if numInactive3 != numInactive2+1 {
		t.Errorf("Number of inactive peers unexpected: wanted %d, found %d", numInactive2+1, numInactive3)
	}
	if numActive3 != numActive2-1 {
		t.Errorf("Number of active peers unexpected: wanted %d, found %d", numActive2-1, numActive3)
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
	if err := genesisState.SetSlot(111); err != nil {
		t.Fatal(err)
	}
	if err := genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  finalizedRoot[:],
	}

	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		ctx: context.Background(),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		digest, err := r.forkDigest()
		if err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			ForkDigest:     digest[:],
			HeadSlot:       genesisState.Slot(),
			HeadRoot:       headRoot[:],
			FinalizedEpoch: 5,
			FinalizedRoot:  finalizedRoot[:],
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

	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to continue being connected")
	}
}

func TestStatusRPCRequest_FinalizedBlockExists(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	db, _ := testingDB.SetupDB(t)

	// Set up a head state with data we expect.
	headRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 111})
	if err != nil {
		t.Fatal(err)
	}
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: blkSlot})
	if err != nil {
		t.Fatal(err)
	}
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if err := genesisState.SetSlot(111); err != nil {
		t.Fatal(err)
	}
	if err := genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: blkSlot}}); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveGenesisBlockRoot(context.Background(), finalizedRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 3,
		Root:  finalizedRoot[:],
	}
	totalSec := params.BeaconConfig().SlotsPerEpoch * 5 * params.BeaconConfig().SecondsPerSlot
	genTime := time.Now().Unix() - int64(totalSec)
	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Unix(genTime, 0),
			ValidatorsRoot: [32]byte{'A'},
		},
		ctx: context.Background(),
	}

	r2 := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Unix(genTime, 0),
			ValidatorsRoot: [32]byte{'A'},
		},
		db:  db,
		ctx: context.Background(),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		err := r2.validateStatusMessage(context.Background(), out)
		if err != nil {
			t.Fatal(err)
		}
	})

	p1.AddConnectionHandler(r.sendRPCStatusRequest)
	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, 100*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	if len(p1.BHost.Network().Peers()) != 1 {
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
	if err := genesisState.SetSlot(111); err != nil {
		t.Fatal(err)
	}
	if err := genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  finalizedRoot[:],
	}

	r := &Service{
		p2p: p1,
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		ctx: context.Background(),
	}

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		if err := r.p2p.Encoding().DecodeWithMaxLength(stream, out); err != nil {
			t.Fatal(err)
		}
		expected := &pb.Status{
			ForkDigest:     []byte{1, 1, 1, 1},
			HeadSlot:       genesisState.Slot(),
			HeadRoot:       headRoot[:],
			FinalizedEpoch: 5,
			FinalizedRoot:  finalizedRoot[:],
		}
		if _, err := stream.Write([]byte{responseCodeSuccess}); err != nil {
			log.WithError(err).Error("Failed to write to stream")
		}
		_, err := r.p2p.Encoding().EncodeWithMaxLength(stream, expected)
		if err != nil {
			t.Errorf("Could not send status: %v", err)
		}
	})

	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	time.Sleep(100 * time.Millisecond)

	connectionState, err := p1.Peers().ConnectionState(p2.PeerID())
	if err != nil {
		t.Fatal("Failed to obtain peer connection state")
	}
	if connectionState != peers.PeerDisconnected {
		t.Error("Expected peer to be disconnected")
	}

	badResponses, err := p1.Peers().Scorer().BadResponses(p2.PeerID())
	if err != nil {
		t.Fatal("Failed to obtain peer connection state")
	}
	if badResponses != 1 {
		t.Errorf("Bad response was not bumped to one, instead it is %d", badResponses)
	}
}

func TestStatusRPC_ValidGenesisMessage(t *testing.T) {
	// Set up a head state with data we expect.
	headRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: 111})
	if err != nil {
		t.Fatal(err)
	}
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := ssz.HashTreeRoot(&ethpb.BeaconBlock{Slot: blkSlot})
	if err != nil {
		t.Fatal(err)
	}
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	if err := genesisState.SetSlot(111); err != nil {
		t.Fatal(err)
	}
	if err := genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot); err != nil {
		t.Fatal(err)
	}
	finalizedCheckpt := &ethpb.Checkpoint{
		Epoch: 5,
		Root:  finalizedRoot[:],
	}
	r := &Service{
		chain: &mock.ChainService{
			State:               genesisState,
			FinalizedCheckPoint: finalizedCheckpt,
			Root:                headRoot[:],
			Fork: &pb.Fork{
				PreviousVersion: params.BeaconConfig().GenesisForkVersion,
				CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			},
			Genesis:        time.Now(),
			ValidatorsRoot: [32]byte{'A'},
		},
		ctx: context.Background(),
	}
	digest, err := r.forkDigest()
	if err != nil {
		t.Fatal(err)
	}
	// There should be no error for a status message
	// with a genesis checkpoint.
	err = r.validateStatusMessage(r.ctx, &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  params.BeaconConfig().ZeroHash[:],
		FinalizedEpoch: 0,
		HeadRoot:       headRoot[:],
		HeadSlot:       111,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestShouldResync(t *testing.T) {
	type args struct {
		genesis     time.Time
		syncing     bool
		headSlot    uint64
		genesisTime uint64
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "genesis epoch should not resync when syncing is true",
			args: args{
				headSlot: uint64(31),
				genesis:  roughtime.Now(),
				syncing:  true,
			},
			want: false,
		},
		{
			name: "genesis epoch should not resync when syncing is false",
			args: args{
				headSlot: uint64(31),
				genesis:  roughtime.Now(),
				syncing:  false,
			},
			want: false,
		},
		{
			name: "two epochs behind, resync ok",
			args: args{
				headSlot: uint64(31),
				genesis:  roughtime.Now().Add(-1 * 96 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second),
				syncing:  false,
			},
			want: true,
		},
		{
			name: "two epochs behind, already syncing",
			args: args{
				headSlot: uint64(31),
				genesis:  roughtime.Now().Add(-1 * 96 * time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second),
				syncing:  true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		headState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
		if err != nil {
			t.Fatal(err)
		}
		if err := headState.SetSlot(tt.args.headSlot); err != nil {
			t.Fatal(err)
		}
		r := &Service{
			chain: &mock.ChainService{
				State:   headState,
				Genesis: tt.args.genesis,
			},
			ctx:         context.Background(),
			initialSync: &mockSync.Sync{IsSyncing: tt.args.syncing},
		}
		t.Run(tt.name, func(t *testing.T) {
			if got := r.shouldReSync(); got != tt.want {
				t.Errorf("shouldReSync() = %v, want %v", got, tt.want)
			}
		})
	}
}
