package db

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestNilDB(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	b := types.NewBlock(nil)
	h, _ := b.Hash()

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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	b1 := types.NewBlock(nil)
	h1, _ := b1.Hash()

	err := db.SaveBlock(b1)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b1Prime, err := db.GetBlock(h1)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h1Prime, _ := b1Prime.Hash()

	if b1Prime == nil || h1 != h1Prime {
		t.Fatalf("get should return b1: %x", h1)
	}

	b2 := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	h2, _ := b2.Hash()

	err = db.SaveBlock(b2)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b2Prime, err := db.GetBlock(h2)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h2Prime, _ := b2Prime.Hash()
	if b2Prime == nil || h2 != h2Prime {
		t.Fatalf("get should return b2: %x", h2)
	}
}

func TestGetBlockBySlotEmptyChain(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	b, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Errorf("failure when fetching block by slot: %v", err)
	}
	if b != nil {
		t.Error("GetBlockBySlot should return nil for an empty chain")
	}
}

func TestUpdateChainHeadNoBlock(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}
	aState, err := db.GetActiveState()
	if err != nil {
		t.Fatalf("failed to get active state: %v", err)
	}

	b := types.NewBlock(&pb.BeaconBlock{Slot: 1})
	if err := db.UpdateChainHead(b, aState, nil); err == nil {
		t.Fatalf("expected UpdateChainHead to fail if the block does not exist: %v", err)
	}
}

func TestUpdateChainHead(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	b, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Fatalf("failed to get genesis block: %v", err)
	}
	bHash, err := b.Hash()
	if err != nil {
		t.Fatalf("failed to get hash of b: %v", err)
	}

	aState, err := db.GetActiveState()
	if err != nil {
		t.Fatalf("failed to get active state: %v", err)
	}

	b2 := types.NewBlock(&pb.BeaconBlock{
		Slot:           1,
		AncestorHashes: [][]byte{bHash[:]},
	})
	b2Hash, err := b2.Hash()
	if err != nil {
		t.Fatalf("failed to hash b2: %v", err)
	}
	if err := db.SaveBlock(b2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(b2, aState, nil); err != nil {
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

	b2PrimeHash, err := b2Prime.Hash()
	if err != nil {
		t.Fatalf("failed to hash b2Prime: %v", err)
	}
	b2SigmaHash, err := b2Sigma.Hash()
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
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	err := db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	aState, err := db.GetActiveState()
	if err != nil {
		t.Fatalf("Failed to get active state: %v", err)
	}
	cState, err := db.GetCrystallizedState()
	if err != nil {
		t.Fatalf("Failed to get crystallized state: %v", err)
	}
	cycleLength := params.GetConfig().CycleLength

	b1 := types.NewBlock(&pb.BeaconBlock{Slot: 1})
	if err := db.SaveBlock(b1); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(b1, aState, nil); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err := db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.SlotNumber() != b1.SlotNumber() {
		t.Fatalf("expected height to equal %d, got %d", b1.SlotNumber(), heighestBlock.SlotNumber())
	}

	b2 := types.NewBlock(&pb.BeaconBlock{Slot: cycleLength})
	if err := db.SaveBlock(b2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(b2, aState, nil); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err = db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if heighestBlock.SlotNumber() != b2.SlotNumber() {
		t.Fatalf("expected height to equal %d, got %d", b2.SlotNumber(), heighestBlock.SlotNumber())
	}

	b3 := types.NewBlock(&pb.BeaconBlock{Slot: 3})
	if err := db.SaveBlock(b3); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(b3, aState, cState); err != nil {
		t.Fatalf("failed to update head: %v", err)
	}
	heighestBlock, err = db.GetChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.SlotNumber() != b3.SlotNumber() {
		t.Fatalf("expected height to equal %d, got %d", b3.SlotNumber(), heighestBlock.SlotNumber())
	}
}

func TestGetGenesisTime(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)

	time, err := db.GetGenesisTime()
	if err == nil {
		t.Fatal("expected GetGenesisTime to fail")
	}

	err = db.InitializeState(nil)
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	time, err = db.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed on second attempt: %v", err)
	}
	time2, err := db.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed on second attempt: %v", err)
	}

	if time != time2 {
		t.Fatalf("Expected %v and %v to be equal", time, time2)
	}
}
