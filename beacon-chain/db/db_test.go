package db

import (
	"os"
	"path"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func setupDB(t *testing.T) *DB {
	// TODO: Fix so that db is always created in the same location
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to open dir: %v", err)
	}

	datadir := path.Join(dir, "test")
	if err := os.RemoveAll(datadir); err != nil {
		t.Fatalf("failed to clean dir: %v", err)
	}

	if err := os.MkdirAll(datadir, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	datafile := path.Join(datadir, "test.db")
	boltDB, err := bolt.Open(datafile, 0600, nil)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	boltDB.NoSync = true

	return NewDB(boltDB)
}
func TestNilDB(t *testing.T) {
	db := setupDB(t)

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
		t.Fatalf("get should return nil for a non-existant key")
	}
}

func TestSave(t *testing.T) {
	db := setupDB(t)

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
		SlotNumber: 0,
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
	db := setupDB(t)

	b, err := db.GetBlockBySlot(0)
	if err != nil {
		t.Errorf("failure when fetching block by slot: %v", err)
	}
	if b != nil {
		t.Error("GetBlockBySlot should return nil for an empty chain")
	}
}
/*
func TestRecordChainTipNoBlock(t *testing.T) {
	bolt := testutils.SetupDB(t)
	db := NewDB(bolt)

	b, err := types.NewGenesisBlock()
	if err != nil {
		t.Fatalf("failed to initialize genesis block")
	}

	if err = db.RecordChainTip(b); err != nil {
		t.Fatalf("failed to record the new tip of the main chain: %v", err)
	}

	_, err = db.GetBlockBySlot(0)
	if err == nil {
		t.Fatalf("GetBlockBySlot should have failed since the block was never recorded")
	}

	_, err = db.GetChainTip()
	if err == nil {
		t.Fatalf("GetChainTip should have failed since the block was never recorded")
	}
}

func TestChainProgress(t *testing.T) {
	bolt := testutils.SetupDB(t)
	db := NewDB(bolt)

	b, err := types.NewGenesisBlock()
	if err != nil {
		t.Fatalf("failed to initialize genesis block")
	}
	if err = db.RecordChainTip(b); err != nil {
		t.Fatalf("failed to record the new tip of the main chain: %v", err)
	}

	_b2 := &pb.BeaconBlock{ SlotNumber: 5 }
	b2 := types.NewBlock(_b2)
	if err != nil {
		t.Fatalf("failed to instantiate next block: %v", err)
	}
	if err = db.SaveBlock(b2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err = db.RecordChainTip(b2); err != nil {
		t.Fatalf("failed to record the new tip of the main chain: %v", err)
	}

	b3 := types.NewBlock(_b2)
	h3, _ := b3.Hash()
	if err = db.RecordChainTip(b3); err == nil {
		t.Fatalf("recording a block with the same height as the current tip should fail")
	}
	b3Prime, err := db.GetChainTip()
	if err != nil {
		t.Fatalf("failed to get the tip of the main chain: %v", err)
	}
	h3Prime, _ := b3Prime.Hash()
	if h3 != h3Prime {
		t.Fatalf("the tip of the chain should equal the previously recorded block: %x %x", h3, h3Prime)
	}
}
*/
