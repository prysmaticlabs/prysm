package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/prysmaticlabs/go-ssz"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestRecentBeaconBlocksRPCHandler_ReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d := db.SetupDB(t)
	defer db.TeardownDB(t, d)

	var blkRoots [][]byte
	// Populate the database with blocks that would match the request.
	for i := 1; i < 11; i++ {
		blk := &ethpb.BeaconBlock{
			Slot: uint64(i),
		}
		root, err := ssz.SigningRoot(blk)
		if err != nil {
			t.Fatal(err)
		}
		if err := d.SaveBlock(context.Background(), blk); err != nil {
			t.Fatal(err)
		}
		blkRoots = append(blkRoots, root[:])
	}
	req := &pb.RecentBeaconBlocksRequest{
		BlockRoots: blkRoots,
	}

	r := &RegularSync{p2p: p1, db: d}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		res := &pb.BeaconBlocksResponse{}
		if err := r.p2p.Encoding().Decode(stream, res); err != nil {
			t.Error(err)
		}
		if len(res.Blocks) != len(req.BlockRoots) {
			t.Errorf("Received only %d blocks, expected %d", len(res.Blocks), len(req.BlockRoots))
		}
		for i, blk := range res.Blocks {
			if blk.Slot != uint64(i+1) {
				t.Errorf("Received unexpected block slot %d but wanted %d", blk.Slot, i+1)
			}
		}
	})

	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.recentBeaconBlocksRPCHandler(context.Background(), req, stream1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}
