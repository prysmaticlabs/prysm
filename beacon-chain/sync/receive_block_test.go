package sync

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// totalMissingParents describes the number of missing parent requests we want to test.
var totalMissingParents = 50

func setupBlockParents(t *testing.T, genesisRoot [32]byte) ([]*pb.BeaconBlock, [][32]byte) {
	parents := []*pb.BeaconBlock{}
	parentRoots := [][32]byte{}
	// Sets up a list of block parents of the form:
	//   Parent 1: {Slot: 1, Parent: genesisBlock},
	//   Parent 2: {Slot: 3, Parent: Parent1},
	//   Parent 3: {Slot: 5, Parent: parent2},
	//   ...
	for slot := 1; slot < totalMissingParents; slot += 2 {
		parent := &pb.BeaconBlock{
			Slot: uint64(slot),
		}
		// At slot 1, the parent is the genesis block.
		if slot == 1 {
			parent.ParentRootHash32 = genesisRoot[:]
		} else {
			parent.ParentRootHash32 = parentRoots[len(parentRoots)-1][:]
		}
		parentRoot, err := hashutil.HashBeaconBlock(parent)
		if err != nil {
			t.Fatal(err)
		}
		parents = append(parents, parent)
		parentRoots = append(parentRoots, parentRoot)
	}
	return parents, parentRoots
}

func setupBlocksMissingParent(parents []*pb.BeaconBlock, parentRoots [][32]byte) []*pb.BeaconBlock {
	blocksMissingParent := []*pb.BeaconBlock{}
	// Sets up a list of block with missing parents of the form:
	//   Parent 1: {Slot: 6, Parent: parents[0]},
	//   Parent 2: {Slot: 4, Parent: parents[1]},
	//   Parent 3: {Slot: 2, Parent: parents[2]},
	//   ...
	for slot := parents[len(parents)-1].Slot + 1; slot >= 2; slot -= 2 {
		blocksMissingParent = append(blocksMissingParent, &pb.BeaconBlock{
			Slot: slot,
		})
	}
	for i := range parentRoots {
		blocksMissingParent[i].ParentRootHash32 = parentRoots[i][:]
	}
	return blocksMissingParent
}

func TestReceiveBlockAnnounce_SkipsBlacklistedBlock(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	rsCfg := DefaultRegularSyncConfig()
	rsCfg.ChainService = &mockChainService{
		db: db,
	}
	rsCfg.BeaconDB = db
	rs := NewRegularSyncService(context.Background(), rsCfg)
	evilBlockHash := []byte("evil-block")
	blockRoot := bytesutil.ToBytes32(evilBlockHash)
	db.MarkEvilBlockHash(blockRoot)
	msg := p2p.Message{
		Ctx: context.Background(),
		Data: &pb.BeaconBlockAnnounce{
			Hash: blockRoot[:],
		},
	}
	if err := rs.receiveBlockAnnounce(msg); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	testutil.AssertLogsContain(t, hook, "Received blacklisted block")
	hook.Reset()
}

func TestReceiveBlock_RecursivelyProcessesChildren(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

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
	if err := db.SaveState(ctx, genesisState); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateChainHead(ctx, genesisBlock, genesisState); err != nil {
		t.Fatal(err)
	}

	parents, parentRoots := setupBlockParents(t, genesisRoot)
	blocksMissingParent := setupBlocksMissingParent(parents, parentRoots)

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
