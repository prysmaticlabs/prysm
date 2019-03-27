package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
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
		Slot: 0,
	}
	genesisRoot, err := hashutil.HashBeaconBlock(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	genesisState := &pb.BeaconState{
		Slot:           0,
		FinalizedEpoch: 0,
	}
	if err := db.SaveBlock(genesisBlock); err != nil {
		t.Fatal(err)
	}
	if err := db.SaveState(genesisState); err != nil {
		t.Fatal(err)
	}

	parent1 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: genesisRoot[:],
	}
	parent1Root, err := hashutil.HashBeaconBlock(parent1)
	if err != nil {
		t.Fatal(err)
	}
	parent2 := &pb.BeaconBlock{
		Slot:             3,
		ParentRootHash32: parent1Root[:],
	}
	parent2Root, err := hashutil.HashBeaconBlock(parent2)
	if err != nil {
		t.Fatal(err)
	}
	parent3 := &pb.BeaconBlock{
		Slot:             5,
		ParentRootHash32: parent2Root[:],
	}
	parent3Root, err := hashutil.HashBeaconBlock(parent3)
	if err != nil {
		t.Fatal(err)
	}
	parents := []*pb.BeaconBlock{parent1, parent2, parent3}

	blocksMissingParent := []*pb.BeaconBlock{
		{
			Slot:             6,
			ParentRootHash32: parent1Root[:],
		},
		{
			Slot:             4,
			ParentRootHash32: parent2Root[:],
		},
		{
			Slot:             2,
			ParentRootHash32: parent3Root[:],
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
