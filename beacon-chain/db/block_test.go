package db

import (
	"testing"

	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestNilDB(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &pb.BeaconBlock{}
	h, _ := b.Hash(block)

	hasBlock := db.HasBlock(h)
	if hasBlock {
		t.Fatal("HashBlock should return false")
	}

	bPrime, err := db.GetBlock(h)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSave(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block1 := &pb.BeaconBlock{}
	h1, _ := b.Hash(block1)

	err := db.SaveBlock(block1)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b1Prime, err := db.GetBlock(h1)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h1Prime, _ := b.Hash(b1Prime)

	if b1Prime == nil || h1 != h1Prime {
		t.Fatalf("get should return b1: %x", h1)
	}

	block2 := &pb.BeaconBlock{
		Slot: 0,
	}
	h2, _ := b.Hash(block2)

	err = db.SaveBlock(block2)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b2Prime, err := db.GetBlock(h2)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h2Prime, _ := b.Hash(b2Prime)
	if b2Prime == nil || h2 != h2Prime {
		t.Fatalf("get should return b2: %x", h2)
	}
}

func TestGetBlockBySlotEmptyChain(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Errorf("failure when fetching block by slot: %v", err)
	}
	if block != nil {
		t.Error("GetBlockBySlot should return nil for an empty chain")
	}
}

func TestUpdateChainHeadNoBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}
	beaconState, err := db.GetState()
	if err != nil {
		t.Fatalf("failed to get beacon state: %v", err)
	}

	block := &pb.BeaconBlock{Slot: 1}
	if err := db.UpdateChainHead(block, beaconState); err == nil {
		t.Fatalf("expected UpdateChainHead to fail if the block does not exist: %v", err)
	}
}

func TestUpdateChainHead(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	block, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Fatalf("failed to get genesis block: %v", err)
	}
	bHash, err := b.Hash(block)
	if err != nil {
		t.Fatalf("failed to get hash of b: %v", err)
	}

	beaconState, err := db.GetState()
	if err != nil {
		t.Fatalf("failed to get beacon state: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: bHash[:],
	}
	b2Hash, err := b.Hash(block2)
	if err != nil {
		t.Fatalf("failed to hash b2: %v", err)
	}
	if err := db.SaveBlock(block2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(block2, beaconState); err != nil {
		t.Fatalf("failed to record the new head of the main chain: %v", err)
	}

	b2Prime, err := db.GetBlockBySlot(1)
	if err != nil {
		t.Fatalf("failed to retrieve slot 1: %v", err)
	}
	b2Sigma, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to retrieve head: %v", err)
	}

	b2PrimeHash, err := b.Hash(b2Prime)
	if err != nil {
		t.Fatalf("failed to hash b2Prime: %v", err)
	}
	b2SigmaHash, err := b.Hash(b2Sigma)
	if err != nil {
		t.Fatalf("failed to hash b2Sigma: %v", err)
	}

	if b2Hash != b2PrimeHash {
		t.Fatalf("expected %x and %x to be equal", b2Hash, b2PrimeHash)
	}
	if b2Hash != b2SigmaHash {
		t.Fatalf("expected %x and %x to be equal", b2Hash, b2SigmaHash)
	}
}

func TestChainProgress(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	beaconState, err := db.GetState()
	if err != nil {
		t.Fatalf("Failed to get beacon state: %v", err)
	}
	cycleLength := params.BeaconConfig().CycleLength

	block1 := &pb.BeaconBlock{Slot: 1}
	if err := db.SaveBlock(block1); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(block1, beaconState); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.GetSlot() != block1.GetSlot() {
		t.Fatalf("expected height to equal %d, got %d", block1.GetSlot(), heighestBlock.GetSlot())
	}

	block2 := &pb.BeaconBlock{Slot: cycleLength}
	if err := db.SaveBlock(block2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(block2, beaconState); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err = db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if heighestBlock.GetSlot() != block2.GetSlot() {
		t.Fatalf("expected height to equal %d, got %d", block2.GetSlot(), heighestBlock.GetSlot())
	}

	block3 := &pb.BeaconBlock{Slot: 3}
	if err := db.SaveBlock(block3); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(block3, beaconState); err != nil {
		t.Fatalf("failed to update head: %v", err)
	}
	heighestBlock, err = db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.GetSlot() != block3.GetSlot() {
		t.Fatalf("expected height to equal %d, got %d", block3.GetSlot(), heighestBlock.GetSlot())
	}
}
