package sync

import (
	"context"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"testing"
)

func TestReceiveBlock_RecursivelyProcessesChildren(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = &mockChainService{
		db: db,
	}
	rsCfg.BeaconDB = db
	rsCfg.P2P = &mockP2P{}
	rs := NewRegularSyncService(context.Background(), rsCfg)
	genesisBlock := &pb.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	genesisRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	genesisState := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot,
		FinalizedEpoch: params.BeaconConfig().GenesisEpoch,
	}
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(genesisState); err != nil {
		t.Fatal(err)
	}

	parents := []*pb.BeaconBlock{
		{
			Slot: params.BeaconConfig().GenesisSlot + 1,
			ParentRootHash32: genesisRoot[:],
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
			Slot: params.BeaconConfig().GenesisSlot + 2,
			ParentRootHash32: parentRoots[0],
		},
	}

	for _, block := range blocksMissingParent {
		msg := p2p.Message{
			Data: &pb.BeaconBlockResponse{
				Block: block,
			},
			Ctx: context.Background(),
		}
		if err := rs.receiveBlock(msg); err != nil {
			t.Fatalf("Could not receive block: %v", err)
		}
	}
	if len(rs.blocksAwaitingProcessing) != len(blocksMissingParent) {
		t.Errorf(
			"Expected blocks awaiting processing map len = %d, received len = %d",
			len(blocksMissingParent),
			len(rs.blocksAwaitingProcessing),
		)
	}
	for _, block := range parents {
		msg := p2p.Message{
			Data: &pb.BeaconBlockResponse{
				Block: block,
			},
			Ctx: context.Background(),
		}
		if err := rs.receiveBlock(msg); err != nil {
			t.Fatalf("Could not receive block: %v", err)
		}
	}
	if len(rs.blocksAwaitingProcessing) > 0 {
		t.Errorf("Expected blocks awaiting processing map to be empty, received len = %d", len(rs.blocksAwaitingProcessing))
	}
}
