package sync

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestReceiveBlock_RecursivelyProcessesChildren(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = &mockChainService{}
	rsCfg.BeaconDB = db
	rsCfg.P2P = &mockP2P{}
	rs := NewRegularSyncService(context.Background(), rsCfg)

	parents := []*pb.BeaconBlock{
		{
			Slot: params.BeaconConfig().GenesisSlot + 9,
		},
		{
			Slot: params.BeaconConfig().GenesisSlot + 19,
		},
	}
    parentRoots := make([][]byte, len(parents))
    for i := range parents {
    	h, err := hashutil.HashBeaconBlock(parents[i])
    	if err != nil {
    		t.Fatal(err)
		}
		parentRoots[i] = h[:]
	}

	blocksMissingParent := []*pb.BeaconBlock{
		{
			Slot: params.BeaconConfig().GenesisSlot + 10,
			ParentRootHash32: parentRoots[0],
		},
		{
			Slot: params.BeaconConfig().GenesisSlot + 20,
			ParentRootHash32: parentRoots[1],
		},
	}

	for _, block := range blocksMissingParent {
		msg := p2p.Message{
			Data: block,
		}
		if err := rs.receiveBlock(msg); err != nil {
			t.Fatalf("Could not receive block: %v", err)
		}
	}
	if len(rs.blocksAwaitingProcessing) > 0 {
		t.Error("Expected blocks awaiting processing map to be empty, received len = %d", len(rs.blocksAwaitingProcessing))
	}
}
