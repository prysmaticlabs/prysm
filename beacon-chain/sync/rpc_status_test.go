package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kevinms/leakybucket-go"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestStatusRPCHandler_Disconnects_OnForkVersionMismatch(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	root := [32]byte{'C'}

	r := &Service{p2p: p1,
		rateLimiter: newRateLimiter(p1),
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
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		if !bytes.Equal(out.FinalizedRoot, root[:]) {
			t.Errorf("Expected finalized root of %#x but got %#x", root, out.FinalizedRoot)
		}
		assert.NoError(t, stream.Close())
	})

	pcl2 := protocol.ID("/eth2/beacon_chain/req/goodbye/1/ssz_snappy")
	topic = string(pcl2)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg2 sync.WaitGroup
	wg2.Add(1)
	p2.BHost.SetStreamHandler(pcl2, func(stream network.Stream) {
		defer wg2.Done()
		msg := new(uint64)
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, msg))
		assert.Equal(t, codeWrongNetwork, *msg)
		assert.NoError(t, stream.Close())
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	assert.NoError(t, r.statusRPCHandler(context.Background(), &pb.Status{ForkDigest: []byte("fake")}, stream1))

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	if testutil.WaitTimeout(&wg2, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	assert.Equal(t, 0, len(p1.BHost.Network().Peers()), "handler did not disconnect peer")
}

func TestStatusRPCHandler_ConnectsOnGenesis(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	root := [32]byte{}

	r := &Service{p2p: p1,
		rateLimiter: newRateLimiter(p1),
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
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)

	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		if !bytes.Equal(out.FinalizedRoot, root[:]) {
			t.Errorf("Expected finalized root of %#x but got %#x", root, out.FinalizedRoot)
		}
	})

	stream1, err := p1.BHost.NewStream(context.Background(), p2.BHost.ID(), pcl)
	require.NoError(t, err)
	digest, err := r.forkDigest()
	require.NoError(t, err)

	err = r.statusRPCHandler(context.Background(), &pb.Status{ForkDigest: digest[:], FinalizedRoot: params.BeaconConfig().ZeroHash[:]}, stream1)
	assert.NoError(t, err)

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Handler disconnected with peer")
}

func TestStatusRPCHandler_ReturnsHelloMessage(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	db, _ := testingDB.SetupDB(t)

	// Set up a head state with data we expect.
	headRoot, err := (&ethpb.BeaconBlock{Slot: 111}).HashTreeRoot()
	require.NoError(t, err)
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := (&ethpb.BeaconBlock{Slot: blkSlot}).HashTreeRoot()
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot))
	require.NoError(t, db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: blkSlot}}))
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), finalizedRoot))
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
		db:          db,
		rateLimiter: newRateLimiter(p1),
	}
	digest, err := r.forkDigest()
	require.NoError(t, err)

	// Setup streams
	pcl := protocol.ID("/testing")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
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
	require.NoError(t, err)

	err = r.statusRPCHandler(context.Background(), &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  finalizedRoot[:],
		FinalizedEpoch: 3,
	}, stream1)
	assert.NoError(t, err)

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
	require.NoError(t, err)
	blk := testutil.NewBeaconBlock()
	blk.Block.Slot = 0
	require.NoError(t, db.SaveBlock(context.Background(), blk))
	finalizedRoot, err := blk.Block.HashTreeRoot()
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), finalizedRoot))
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
		db:          db,
		ctx:         context.Background(),
		rateLimiter: newRateLimiter(p1),
	}
	p1.Digest, err = r.forkDigest()
	require.NoError(t, err)

	r2 := &Service{
		chain: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0, Root: finalizedRoot[:]},
		},
		p2p:         p2,
		rateLimiter: newRateLimiter(p2),
	}
	p2.Digest, err = r.forkDigest()
	require.NoError(t, err)

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		log.WithField("status", out).Warn("received status")
		resp := &pb.Status{HeadSlot: 100, ForkDigest: p2.Digest[:],
			FinalizedRoot: finalizedRoot[:], FinalizedEpoch: 0}
		_, err := stream.Write([]byte{responseCodeSuccess})
		assert.NoError(t, err)
		_, err = r.p2p.Encoding().EncodeWithMaxLength(stream, resp)
		assert.NoError(t, err)
		log.WithField("status", out).Warn("sending status")
		if err := stream.Close(); err != nil {
			t.Log(err)
		}
	})

	pcl = "/eth2/beacon_chain/req/ping/1/ssz_snappy"
	topic = string(pcl)
	r2.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg2 sync.WaitGroup
	wg2.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg2.Done()
		out := new(uint64)
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		assert.Equal(t, uint64(2), *out)
		assert.NoError(t, r2.pingHandler(context.Background(), out, stream))
		assert.NoError(t, stream.Close())
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

	assert.Equal(t, numInactive1, numInactive1, "Number of inactive peers changed unexpectedly")
	assert.Equal(t, numActive1+1, numActive2, "Number of active peers unexpected")

	require.NoError(t, p2.Disconnect(p1.PeerID()))
	p1.Peers().SetConnectionState(p2.PeerID(), peers.PeerDisconnected)

	// Wait for disconnect event to trigger.
	time.Sleep(200 * time.Millisecond)

	numInactive3 := len(p1.Peers().Inactive())
	numActive3 := len(p1.Peers().Active())
	assert.Equal(t, numInactive2+1, numInactive3, "Number of inactive peers unexpected")
	assert.Equal(t, numActive2-1, numActive3, "Number of active peers unexpected")
}

func TestStatusRPCRequest_RequestSent(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)

	// Set up a head state with data we expect.
	headRoot, err := (&ethpb.BeaconBlock{Slot: 111}).HashTreeRoot()
	require.NoError(t, err)
	finalizedRoot, err := (&ethpb.BeaconBlock{Slot: 40}).HashTreeRoot()
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot))
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
		ctx:         context.Background(),
		rateLimiter: newRateLimiter(p1),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		digest, err := r.forkDigest()
		assert.NoError(t, err)
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

	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to continue being connected")
}

func TestStatusRPCRequest_FinalizedBlockExists(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	db, _ := testingDB.SetupDB(t)

	// Set up a head state with data we expect.
	headRoot, err := (&ethpb.BeaconBlock{Slot: 111}).HashTreeRoot()
	require.NoError(t, err)
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := (&ethpb.BeaconBlock{Slot: blkSlot}).HashTreeRoot()
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot))
	require.NoError(t, db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: blkSlot}}))
	require.NoError(t, db.SaveGenesisBlockRoot(context.Background(), finalizedRoot))
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
		ctx:         context.Background(),
		rateLimiter: newRateLimiter(p1),
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
		db:          db,
		ctx:         context.Background(),
		rateLimiter: newRateLimiter(p1),
	}

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
		assert.NoError(t, r2.validateStatusMessage(context.Background(), out))
	})

	p1.AddConnectionHandler(r.sendRPCStatusRequest)
	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, 100*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}

	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to continue being connected")
}

func TestStatusRPCRequest_BadPeerHandshake(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)

	// Set up a head state with data we expect.
	headRoot, err := (&ethpb.BeaconBlock{Slot: 111}).HashTreeRoot()
	require.NoError(t, err)
	finalizedRoot, err := (&ethpb.BeaconBlock{Slot: 40}).HashTreeRoot()
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot))
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
		ctx:         context.Background(),
		rateLimiter: newRateLimiter(p1),
	}

	r.Start()

	// Setup streams
	pcl := protocol.ID("/eth2/beacon_chain/req/status/1/ssz_snappy")
	topic := string(pcl)
	r.rateLimiter.limiterMap[topic] = leakybucket.NewCollector(1, 1, false)
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		out := &pb.Status{}
		assert.NoError(t, r.p2p.Encoding().DecodeWithMaxLength(stream, out))
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
		assert.NoError(t, err)
	})

	p1.Connect(p2)

	if testutil.WaitTimeout(&wg, time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
	time.Sleep(100 * time.Millisecond)

	connectionState, err := p1.Peers().ConnectionState(p2.PeerID())
	require.NoError(t, err, "Failed to obtain peer connection state")
	assert.Equal(t, peers.PeerDisconnected, connectionState, "Expected peer to be disconnected")

	badResponses, err := p1.Peers().Scorers().BadResponsesScorer().Count(p2.PeerID())
	require.NoError(t, err, "Failed to obtain peer connection state")
	assert.Equal(t, 1, badResponses, "Bad response was not bumped to one")
}

func TestStatusRPC_ValidGenesisMessage(t *testing.T) {
	// Set up a head state with data we expect.
	headRoot, err := (&ethpb.BeaconBlock{Slot: 111}).HashTreeRoot()
	require.NoError(t, err)
	blkSlot := 3 * params.BeaconConfig().SlotsPerEpoch
	finalizedRoot, err := (&ethpb.BeaconBlock{Slot: blkSlot}).HashTreeRoot()
	require.NoError(t, err)
	genesisState, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	require.NoError(t, err)
	require.NoError(t, genesisState.SetSlot(111))
	require.NoError(t, genesisState.UpdateBlockRootAtIndex(111%params.BeaconConfig().SlotsPerHistoricalRoot, headRoot))
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
	require.NoError(t, err)
	// There should be no error for a status message
	// with a genesis checkpoint.
	err = r.validateStatusMessage(r.ctx, &pb.Status{
		ForkDigest:     digest[:],
		FinalizedRoot:  params.BeaconConfig().ZeroHash[:],
		FinalizedEpoch: 0,
		HeadRoot:       headRoot[:],
		HeadSlot:       111,
	})
	require.NoError(t, err)
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
		require.NoError(t, err)
		require.NoError(t, headState.SetSlot(tt.args.headSlot))
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
