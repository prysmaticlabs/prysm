package sync

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

//    /- b1 - b2
// b0
//    \- b3
// Test b1 was missing then received and we can process b0 -> b1 -> b2
func TestRegularSyncBeaconBlockSubscriber_ProcessPendingBlocks1(t *testing.T) {
	db, _ := dbtest.SetupDB(t)

	p1 := p2ptest.NewTestP2P(t)
	r := &Service{
		p2p: p1,
		db:  db,
		chain: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}
	err := r.initCaches()
	require.NoError(t, err)

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b0))
	b0Root, err := stateutil.BlockRoot(b0.Block)
	require.NoError(t, err)
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b3))
	// Incomplete block link
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	b1Root, err := stateutil.BlockRoot(b1.Block)
	require.NoError(t, err)
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}}
	b2Root, err := stateutil.BlockRoot(b1.Block)
	require.NoError(t, err)

	// Add b2 to the cache
	r.slotToPendingBlocks[b2.Block.Slot] = b2
	r.seenPendingBlocks[b2Root] = true

	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 1, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 1, len(r.seenPendingBlocks), "Incorrect size for seen pending block")

	// Add b1 to the cache
	r.slotToPendingBlocks[b1.Block.Slot] = b1
	r.seenPendingBlocks[b1Root] = true
	require.NoError(t, r.db.SaveBlock(context.Background(), b1))
	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 0, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 0, len(r.seenPendingBlocks), "Incorrect size for seen pending block")
}

//    /- b1 - b2 - b5
// b0
//    \- b3 - b4
// Test b2 and b3 were missed, after receiving them we can process 2 chains.
func TestRegularSyncBeaconBlockSubscriber_ProcessPendingBlocks2(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")
	pcl := protocol.ID("/eth2/beacon_chain/req/hello/1/ssz_snappy")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		code, errMsg, err := ReadStatusCode(stream, p1.Encoding())
		assert.NoError(t, err)
		if code == 0 {
			t.Error("Expected a non-zero code")
		}
		if errMsg != errWrongForkDigestVersion.Error() {
			t.Logf("Received error string len %d, wanted error string len %d", len(errMsg), len(errWrongForkDigestVersion.Error()))
			t.Errorf("Received unexpected message response in the stream: %s. Wanted %s.", errMsg, errWrongForkDigestVersion.Error())
		}
	})

	r := &Service{
		p2p: p1,
		db:  db,
		chain: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}
	err := r.initCaches()
	require.NoError(t, err)
	p1.Peers().Add(new(enr.Record), p2.PeerID(), nil, network.DirOutbound)
	p1.Peers().SetConnectionState(p2.PeerID(), peers.PeerConnected)
	p1.Peers().SetChainState(p2.PeerID(), &pb.Status{})

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b0))
	b0Root, err := stateutil.BlockRoot(b0.Block)
	require.NoError(t, err)
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b1))
	b1Root, err := stateutil.BlockRoot(b1.Block)
	require.NoError(t, err)

	// Incomplete block links
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}
	b2Root, err := b2.HashTreeRoot()
	require.NoError(t, err)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: b2Root[:]}
	b5Root, err := b5.HashTreeRoot()
	require.NoError(t, err)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}
	b3Root, err := b3.HashTreeRoot()
	require.NoError(t, err)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: b3Root[:]}
	b4Root, err := b4.HashTreeRoot()
	require.NoError(t, err)

	r.slotToPendingBlocks[b4.Slot] = &ethpb.SignedBeaconBlock{Block: b4}
	r.seenPendingBlocks[b4Root] = true
	r.slotToPendingBlocks[b5.Slot] = &ethpb.SignedBeaconBlock{Block: b5}
	r.seenPendingBlocks[b5Root] = true

	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 2, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 2, len(r.seenPendingBlocks), "Incorrect size for seen pending block")

	// Add b3 to the cache
	r.slotToPendingBlocks[b3.Slot] = &ethpb.SignedBeaconBlock{Block: b3}
	r.seenPendingBlocks[b3Root] = true
	require.NoError(t, r.db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b3}))
	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 1, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 1, len(r.seenPendingBlocks), "Incorrect size for seen pending block")

	// Add b2 to the cache
	r.slotToPendingBlocks[b2.Slot] = &ethpb.SignedBeaconBlock{Block: b2}
	r.seenPendingBlocks[b2Root] = true

	require.NoError(t, r.db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b2}))
	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 0, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 0, len(r.seenPendingBlocks), "Incorrect size for seen pending block")
}

func TestRegularSyncBeaconBlockSubscriber_PruneOldPendingBlocks(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	assert.Equal(t, 1, len(p1.BHost.Network().Peers()), "Expected peers to be connected")

	r := &Service{
		p2p: p1,
		db:  db,
		chain: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 1,
			},
		},
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
		seenPendingBlocks:   make(map[[32]byte]bool),
	}
	err := r.initCaches()
	require.NoError(t, err)
	p1.Peers().Add(new(enr.Record), p1.PeerID(), nil, network.DirOutbound)
	p1.Peers().SetConnectionState(p1.PeerID(), peers.PeerConnected)
	p1.Peers().SetChainState(p1.PeerID(), &pb.Status{})

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b0))
	b0Root, err := stateutil.BlockRoot(b0.Block)
	require.NoError(t, err)
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	require.NoError(t, r.db.SaveBlock(context.Background(), b1))
	b1Root, err := stateutil.BlockRoot(b1.Block)
	require.NoError(t, err)

	// Incomplete block links
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}
	b2Root, err := b2.HashTreeRoot()
	require.NoError(t, err)
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: b2Root[:]}
	b5Root, err := b5.HashTreeRoot()
	require.NoError(t, err)
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}
	b3Root, err := b3.HashTreeRoot()
	require.NoError(t, err)
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: b3Root[:]}
	b4Root, err := b4.HashTreeRoot()
	require.NoError(t, err)

	r.slotToPendingBlocks[b2.Slot] = &ethpb.SignedBeaconBlock{Block: b2}
	r.seenPendingBlocks[b2Root] = true
	r.slotToPendingBlocks[b3.Slot] = &ethpb.SignedBeaconBlock{Block: b3}
	r.seenPendingBlocks[b3Root] = true
	r.slotToPendingBlocks[b4.Slot] = &ethpb.SignedBeaconBlock{Block: b4}
	r.seenPendingBlocks[b4Root] = true
	r.slotToPendingBlocks[b5.Slot] = &ethpb.SignedBeaconBlock{Block: b5}
	r.seenPendingBlocks[b5Root] = true

	require.NoError(t, r.processPendingBlocks(context.Background()))
	assert.Equal(t, 0, len(r.slotToPendingBlocks), "Incorrect size for slot to pending blocks cache")
	assert.Equal(t, 0, len(r.seenPendingBlocks), "Incorrect size for seen pending block")
}

func TestService_sortedPendingSlots(t *testing.T) {
	r := &Service{
		slotToPendingBlocks: make(map[uint64]*ethpb.SignedBeaconBlock),
	}

	var lastSlot uint64 = math.MaxUint64
	r.slotToPendingBlocks[lastSlot] = &ethpb.SignedBeaconBlock{}
	r.slotToPendingBlocks[lastSlot-3] = &ethpb.SignedBeaconBlock{}
	r.slotToPendingBlocks[lastSlot-5] = &ethpb.SignedBeaconBlock{}
	r.slotToPendingBlocks[lastSlot-2] = &ethpb.SignedBeaconBlock{}

	want := []uint64{lastSlot - 5, lastSlot - 3, lastSlot - 2, lastSlot}
	assert.DeepEqual(t, want, r.sortedPendingSlots(), "Unexpected pending slots list")
}
