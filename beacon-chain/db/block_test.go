package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestNilDB_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &pb.BeaconBlock{}
	h, _ := hashutil.HashBeaconBlock(block)

	hasBlock := db.HasBlock(h)
	if hasBlock {
		t.Fatal("HashBlock should return false")
	}

	bPrime, err := db.Block(h)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if bPrime != nil {
		t.Fatalf("get should return nil for a non existent key")
	}
}

func TestSaveBlock_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block1 := &pb.BeaconBlock{}
	h1, _ := hashutil.HashBeaconBlock(block1)

	err := db.SaveBlock(block1)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b1Prime, err := db.Block(h1)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h1Prime, _ := hashutil.HashBeaconBlock(b1Prime)

	if b1Prime == nil || h1 != h1Prime {
		t.Fatalf("get should return b1: %x", h1)
	}

	block2 := &pb.BeaconBlock{
		Slot: 0,
	}
	h2, _ := hashutil.HashBeaconBlock(block2)

	err = db.SaveBlock(block2)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	b2Prime, err := db.Block(h2)
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	h2Prime, _ := hashutil.HashBeaconBlock(b2Prime)
	if b2Prime == nil || h2 != h2Prime {
		t.Fatalf("get should return b2: %x", h2)
	}
}

func TestDeleteBlock_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &pb.BeaconBlock{Slot: params.BeaconConfig().GenesisSlot}
	h, _ := hashutil.HashBeaconBlock(block)

	err := db.SaveBlock(block)
	if err != nil {
		t.Fatalf("save block failed: %v", err)
	}

	savedBlock, err := db.Block(h)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(block, savedBlock) {
		t.Fatal(err)
	}
	if err := db.DeleteBlock(block); err != nil {
		t.Fatal(err)
	}
	savedBlock, err = db.Block(h)
	if err != nil {
		t.Fatal(err)
	}
	if savedBlock != nil {
		t.Errorf("Expected block to have been deleted, received: %v", savedBlock)
	}
}

func TestBlockBySlotEmptyChain_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	block, _ := db.BlockBySlot(ctx, 0)
	if block != nil {
		t.Error("BlockBySlot should return nil for an empty chain")
	}
}

func TestUpdateChainHead_NoBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{})
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}
	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("failed to get beacon state: %v", err)
	}

	block := &pb.BeaconBlock{Slot: 1}
	if err := db.UpdateChainHead(ctx, block, beaconState); err == nil {
		t.Fatalf("expected UpdateChainHead to fail if the block does not exist: %v", err)
	}
}

func TestUpdateChainHead_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{})
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	block, err := db.BlockBySlot(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get genesis block: %v", err)
	}
	bHash, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		t.Fatalf("failed to get hash of b: %v", err)
	}

	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("failed to get beacon state: %v", err)
	}

	block2 := &pb.BeaconBlock{
		Slot:             1,
		ParentRootHash32: bHash[:],
	}
	b2Hash, err := hashutil.HashBeaconBlock(block2)
	if err != nil {
		t.Fatalf("failed to hash b2: %v", err)
	}
	if err := db.SaveBlock(block2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(ctx, block2, beaconState); err != nil {
		t.Fatalf("failed to record the new head of the main chain: %v", err)
	}

	b2Prime, err := db.BlockBySlot(ctx, 1)
	if err != nil {
		t.Fatalf("failed to retrieve slot 1: %v", err)
	}
	b2Sigma, err := db.ChainHead()
	if err != nil {
		t.Fatalf("failed to retrieve head: %v", err)
	}

	b2PrimeHash, err := hashutil.HashBeaconBlock(b2Prime)
	if err != nil {
		t.Fatalf("failed to hash b2Prime: %v", err)
	}
	b2SigmaHash, err := hashutil.HashBeaconBlock(b2Sigma)
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

func TestChainProgress_OK(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	ctx := context.Background()

	genesisTime := uint64(time.Now().Unix())
	deposits, _ := setupInitialDeposits(t, 10)
	err := db.InitializeState(context.Background(), genesisTime, deposits, &pb.Eth1Data{})
	if err != nil {
		t.Fatalf("failed to initialize state: %v", err)
	}

	beaconState, err := db.HeadState(ctx)
	if err != nil {
		t.Fatalf("Failed to get beacon state: %v", err)
	}
	cycleLength := params.BeaconConfig().SlotsPerEpoch

	block1 := &pb.BeaconBlock{Slot: 1}
	if err := db.SaveBlock(block1); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(ctx, block1, beaconState); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err := db.ChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.Slot != block1.Slot {
		t.Fatalf("expected height to equal %d, got %d", block1.Slot, heighestBlock.Slot)
	}

	block2 := &pb.BeaconBlock{Slot: cycleLength}
	if err := db.SaveBlock(block2); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(ctx, block2, beaconState); err != nil {
		t.Fatalf("failed to record the new head: %v", err)
	}
	heighestBlock, err = db.ChainHead()
	if err != nil {
		t.Fatalf("failed to get block: %v", err)
	}
	if heighestBlock.Slot != block2.Slot {
		t.Fatalf("expected height to equal %d, got %d", block2.Slot, heighestBlock.Slot)
	}

	block3 := &pb.BeaconBlock{Slot: 3}
	if err := db.SaveBlock(block3); err != nil {
		t.Fatalf("failed to save block: %v", err)
	}
	if err := db.UpdateChainHead(ctx, block3, beaconState); err != nil {
		t.Fatalf("failed to update head: %v", err)
	}
	heighestBlock, err = db.ChainHead()
	if err != nil {
		t.Fatalf("failed to get chain head: %v", err)
	}
	if heighestBlock.Slot != block3.Slot {
		t.Fatalf("expected height to equal %d, got %d", block3.Slot, heighestBlock.Slot)
	}
}

func TestJustifiedBlock_NoneExists(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	wanted := "no justified block saved"
	_, err := db.JustifiedBlock()
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestJustifiedBlock_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	blkSlot := uint64(10)
	block1 := &pb.BeaconBlock{
		Slot: blkSlot,
	}

	if err := db.SaveJustifiedBlock(block1); err != nil {
		t.Fatalf("could not save justified block: %v", err)
	}

	justifiedBlk, err := db.JustifiedBlock()
	if err != nil {
		t.Fatalf("could not get justified block: %v", err)
	}
	if justifiedBlk.Slot != blkSlot {
		t.Errorf("Saved block does not have the slot from which it was requested, wanted: %d, got: %d",
			blkSlot, justifiedBlk.Slot)
	}
}

func TestFinalizedBlock_NoneExists(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)
	wanted := "no finalized block saved"
	_, err := db.FinalizedBlock()
	if !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected: %s, received: %s", wanted, err.Error())
	}
}

func TestFinalizedBlock_CanSaveRetrieve(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	blkSlot := uint64(22)
	block1 := &pb.BeaconBlock{
		Slot: blkSlot,
	}

	if err := db.SaveFinalizedBlock(block1); err != nil {
		t.Fatalf("could not save finalized block: %v", err)
	}

	finalizedblk, err := db.FinalizedBlock()
	if err != nil {
		t.Fatalf("could not get finalized block: %v", err)
	}
	if finalizedblk.Slot != blkSlot {
		t.Errorf("Saved block does not have the slot from which it was requested, wanted: %d, got: %d",
			blkSlot, finalizedblk.Slot)
	}
}

func TestHasBlock_returnsTrue(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &pb.BeaconBlock{
		Slot: uint64(44),
	}

	root, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if !db.HasBlock(root) {
		t.Fatal("db.HasBlock returned false for block just saved")
	}
}

func TestHighestBlockSlot_UpdatedOnSaveBlock(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	block := &pb.BeaconBlock{
		Slot: 23,
	}

	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if db.HighestBlockSlot() != block.Slot {
		t.Errorf("Unexpected highest slot %d, wanted %d", db.HighestBlockSlot(), block.Slot)
	}

	block.Slot = 55
	if err := db.SaveBlock(block); err != nil {
		t.Fatal(err)
	}

	if db.HighestBlockSlot() != block.Slot {
		t.Errorf("Unexpected highest slot %d, wanted %d", db.HighestBlockSlot(), block.Slot)
	}

}
