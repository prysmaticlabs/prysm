package sync

import (
	"context"
	"math"
	"reflect"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := r.db.SaveBlock(context.Background(), b0); err != nil {
		t.Fatal(err)
	}
	b0Root, err := stateutil.BlockRoot(b0.Block)
	if err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}}
	if err := r.db.SaveBlock(context.Background(), b3); err != nil {
		t.Fatal(err)
	}
	// Incomplete block link
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	b1Root, err := stateutil.BlockRoot(b1.Block)
	if err != nil {
		t.Fatal(err)
	}
	b2 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}}
	b2Root, err := stateutil.BlockRoot(b1.Block)
	if err != nil {
		t.Fatal(err)
	}

	// Add b2 to the cache
	r.slotToPendingBlocks[b2.Block.Slot] = b2
	r.seenPendingBlocks[b2Root] = true

	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 1 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	if len(r.seenPendingBlocks) != 1 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}

	// Add b1 to the cache
	r.slotToPendingBlocks[b1.Block.Slot] = b1
	r.seenPendingBlocks[b1Root] = true
	if err := r.db.SaveBlock(context.Background(), b1); err != nil {
		t.Fatal(err)
	}
	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 0 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	if len(r.seenPendingBlocks) != 0 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}
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
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	pcl := protocol.ID("/eth2/beacon_chain/req/hello/1/ssz")
	var wg sync.WaitGroup
	wg.Add(1)
	p2.BHost.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		code, errMsg, err := ReadStatusCode(stream, p1.Encoding())
		if err != nil {
			t.Fatal(err)
		}
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
	p1.Peers().Add(new(enr.Record), p2.PeerID(), nil, network.DirOutbound)
	p1.Peers().SetConnectionState(p2.PeerID(), peers.PeerConnected)
	p1.Peers().SetChainState(p2.PeerID(), &pb.Status{})

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := r.db.SaveBlock(context.Background(), b0); err != nil {
		t.Fatal(err)
	}
	b0Root, err := stateutil.BlockRoot(b0.Block)
	if err != nil {
		t.Fatal(err)
	}
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	if err := r.db.SaveBlock(context.Background(), b1); err != nil {
		t.Fatal(err)
	}
	b1Root, err := stateutil.BlockRoot(b1.Block)
	if err != nil {
		t.Fatal(err)
	}

	// Incomplete block links
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}
	b2Root, err := ssz.HashTreeRoot(b2)
	if err != nil {
		t.Fatal(err)
	}
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: b2Root[:]}
	b5Root, err := ssz.HashTreeRoot(b5)
	if err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}
	b3Root, err := ssz.HashTreeRoot(b3)
	if err != nil {
		t.Fatal(err)
	}
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: b3Root[:]}
	b4Root, err := ssz.HashTreeRoot(b4)
	if err != nil {
		t.Fatal(err)
	}

	r.slotToPendingBlocks[b4.Slot] = &ethpb.SignedBeaconBlock{Block: b4}
	r.seenPendingBlocks[b4Root] = true
	r.slotToPendingBlocks[b5.Slot] = &ethpb.SignedBeaconBlock{Block: b5}
	r.seenPendingBlocks[b5Root] = true

	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 2 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	if len(r.seenPendingBlocks) != 2 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}

	// Add b3 to the cache
	r.slotToPendingBlocks[b3.Slot] = &ethpb.SignedBeaconBlock{Block: b3}
	r.seenPendingBlocks[b3Root] = true
	if err := r.db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b3}); err != nil {
		t.Fatal(err)
	}
	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 1 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	if len(r.seenPendingBlocks) != 1 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}

	// Add b2 to the cache
	r.slotToPendingBlocks[b2.Slot] = &ethpb.SignedBeaconBlock{Block: b2}
	r.seenPendingBlocks[b2Root] = true

	if err := r.db.SaveBlock(context.Background(), &ethpb.SignedBeaconBlock{Block: b2}); err != nil {
		t.Fatal(err)
	}
	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 0 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	t.Log(r.seenPendingBlocks)
	if len(r.seenPendingBlocks) != 0 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}
}

func TestRegularSyncBeaconBlockSubscriber_PruneOldPendingBlocks(t *testing.T) {
	db, _ := dbtest.SetupDB(t)
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.BHost.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}

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
	p1.Peers().Add(new(enr.Record), p1.PeerID(), nil, network.DirOutbound)
	p1.Peers().SetConnectionState(p1.PeerID(), peers.PeerConnected)
	p1.Peers().SetChainState(p1.PeerID(), &pb.Status{})

	b0 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{}}
	if err := r.db.SaveBlock(context.Background(), b0); err != nil {
		t.Fatal(err)
	}
	b0Root, err := stateutil.BlockRoot(b0.Block)
	if err != nil {
		t.Fatal(err)
	}
	b1 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1, ParentRoot: b0Root[:]}}
	if err := r.db.SaveBlock(context.Background(), b1); err != nil {
		t.Fatal(err)
	}
	b1Root, err := stateutil.BlockRoot(b1.Block)
	if err != nil {
		t.Fatal(err)
	}

	// Incomplete block links
	b2 := &ethpb.BeaconBlock{Slot: 2, ParentRoot: b1Root[:]}
	b2Root, err := ssz.HashTreeRoot(b2)
	if err != nil {
		t.Fatal(err)
	}
	b5 := &ethpb.BeaconBlock{Slot: 5, ParentRoot: b2Root[:]}
	b5Root, err := ssz.HashTreeRoot(b5)
	if err != nil {
		t.Fatal(err)
	}
	b3 := &ethpb.BeaconBlock{Slot: 3, ParentRoot: b0Root[:]}
	b3Root, err := ssz.HashTreeRoot(b3)
	if err != nil {
		t.Fatal(err)
	}
	b4 := &ethpb.BeaconBlock{Slot: 4, ParentRoot: b3Root[:]}
	b4Root, err := ssz.HashTreeRoot(b4)
	if err != nil {
		t.Fatal(err)
	}

	r.slotToPendingBlocks[b2.Slot] = &ethpb.SignedBeaconBlock{Block: b2}
	r.seenPendingBlocks[b2Root] = true
	r.slotToPendingBlocks[b3.Slot] = &ethpb.SignedBeaconBlock{Block: b3}
	r.seenPendingBlocks[b3Root] = true
	r.slotToPendingBlocks[b4.Slot] = &ethpb.SignedBeaconBlock{Block: b4}
	r.seenPendingBlocks[b4Root] = true
	r.slotToPendingBlocks[b5.Slot] = &ethpb.SignedBeaconBlock{Block: b5}
	r.seenPendingBlocks[b5Root] = true

	if err := r.processPendingBlocks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.slotToPendingBlocks) != 0 {
		t.Errorf("Incorrect size for slot to pending blocks cache: got %d", len(r.slotToPendingBlocks))
	}
	if len(r.seenPendingBlocks) != 0 {
		t.Errorf("Incorrect size for seen pending block: got %d", len(r.seenPendingBlocks))
	}
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
	got := r.sortedPendingSlots()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("unexpected pending slots list, want: %v, got: %v", want, got)
	}
}
