package sync

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	db "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	p2ptest "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestBeaconBlocksRPCHandler_ReturnsBlocks(t *testing.T) {
	p1 := p2ptest.NewTestP2P(t)
	p2 := p2ptest.NewTestP2P(t)
	p1.Connect(p2)
	if len(p1.Host.Network().Peers()) != 1 {
		t.Error("Expected peers to be connected")
	}
	d := db.SetupDB(t)
	defer db.TeardownDB(t, d)

	req := &pb.BeaconBlocksRequest{
		HeadSlot: 100,
		Step:     4,
		Count:    100,
	}

	// Populate the database with blocks that would match the request.
	for i := req.HeadSlot; i < req.HeadSlot+(req.Step*req.Count); i++ {
		if err := d.SaveBlock(context.Background(), &ethpb.BeaconBlock{Slot: i}); err != nil {
			t.Fatal(err)
		}
	}

	r := &RegularSync{p2p: p1, db: d}
	pcl := protocol.ID("/testing")

	var wg sync.WaitGroup
	wg.Add(1)
	p2.Host.SetStreamHandler(pcl, func(stream network.Stream) {
		defer wg.Done()
		expectSuccess(t, r, stream)
		res := make([]ethpb.BeaconBlock, 0)
		if err := r.p2p.Encoding().DecodeWithLength(stream, &res); err != nil {
			t.Error(err)
		}
		if uint64(len(res)) != req.Count {
			t.Errorf("Received only %d blocks, expected %d", len(res), req.Count)
		}
		for _, blk := range res {
			if (blk.Slot-req.HeadSlot)%req.Step != 0 {
				t.Errorf("Received unexpected block slot %d", blk.Slot)
			}
		}
	})

	stream1, err := p1.Host.NewStream(context.Background(), p2.Host.ID(), pcl)
	if err != nil {
		t.Fatal(err)
	}

	err = r.beaconBlocksRPCHandler(context.Background(), req, stream1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if testutil.WaitTimeout(&wg, 1*time.Second) {
		t.Fatal("Did not receive stream within 1 sec")
	}
}
