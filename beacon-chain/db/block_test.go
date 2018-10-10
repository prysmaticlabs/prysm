package db

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveAndRemoveBlocks(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:        64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Fatalf("unable to generate hash of block %v", err)
	}

	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("unable to save block %v", err)
	}

	// Adding a different block with the same key
	newblock := types.NewBlock(&pb.BeaconBlock{
		Slot:        4,
		PowChainRef: []byte("b"),
	})

	key := blockKey(hash)
	marshalled, err := proto.Marshal(newblock.Proto())
	if err != nil {
		t.Fatal(err)
	}

	if err := beaconDB.put(key, marshalled); err != nil {
		t.Fatal(err)
	}

	retblock, err := beaconDB.GetBlock(hash)
	if err != nil {
		t.Fatalf("block is unable to be retrieved")
	}

	if retblock.SlotNumber() != newblock.SlotNumber() {
		t.Errorf("slotnumber does not match for saved and retrieved blocks")
	}

	if !bytes.Equal(retblock.PowChainRef().Bytes(), newblock.PowChainRef().Bytes()) {
		t.Errorf("POW chain ref does not match for saved and retrieved blocks")
	}

	if err := beaconDB.removeBlock(hash); err != nil {
		t.Fatalf("error removing block %v", err)
	}

	if _, err := beaconDB.GetBlock(hash); err == nil {
		t.Fatalf("block is able to be retrieved")
	}

	if err := beaconDB.removeBlock(hash); err != nil {
		t.Fatalf("unable to remove block a second time %v", err)
	}
}

func TestCheckBlockBySlotNumber(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:        64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	if err := beaconDB.SaveCanonicalSlotNumber(block.SlotNumber(), hash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("unable to save block %v", err)
	}

	slotExists, err := beaconDB.HasCanonicalBlockForSlot(block.SlotNumber())
	if err != nil {
		t.Fatalf("unable to check for block by slot %v", err)
	}

	if !slotExists {
		t.Error("slot does not exist despite blockhash of canonical block being saved in the db")
	}

	alternateblock := types.NewBlock(&pb.BeaconBlock{
		Slot:        64,
		PowChainRef: []byte("d"),
	})

	althash, err := alternateblock.Hash()
	if err != nil {
		t.Fatalf("unable to hash block %v", err)
	}

	if err := beaconDB.SaveCanonicalSlotNumber(block.SlotNumber(), althash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	retrievedHash, err := beaconDB.get(canonicalBlockKey(block.SlotNumber()))
	if err != nil {
		t.Fatalf("unable to retrieve blockhash %v", err)
	}

	if !bytes.Equal(retrievedHash, althash[:]) {
		t.Errorf("unequal hashes between what was saved and what was retrieved %v, %v", retrievedHash, althash)
	}
}

func TestGetBlockBySlotNumber(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot:        64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	if err := beaconDB.SaveCanonicalSlotNumber(block.SlotNumber(), hash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("unable to save block %v", err)
	}

	retblock, err := beaconDB.GetCanonicalBlockForSlot(block.SlotNumber())
	if err != nil {
		t.Fatalf("unable to get block from db %v", err)
	}

	if !bytes.Equal(retblock.PowChainRef().Bytes(), block.PowChainRef().Bytes()) {
		t.Error("canonical block saved different from block retrieved")
	}

	alternateblock := types.NewBlock(&pb.BeaconBlock{
		Slot:        64,
		PowChainRef: []byte("d"),
	})

	althash, err := alternateblock.Hash()
	if err != nil {
		t.Fatalf("unable to hash block %v", err)
	}

	if err := beaconDB.SaveCanonicalSlotNumber(block.SlotNumber(), althash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if _, err = beaconDB.GetCanonicalBlockForSlot(block.SlotNumber()); err == nil {
		t.Fatal("there should be an error because block does not exist in the db")
	}
}

func TestGetGenesisTime(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	time, err := beaconDB.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed: %v", err)
	}

	time2, err := beaconDB.GetGenesisTime()
	if err != nil {
		t.Fatalf("GetGenesisTime failed on second attempt: %v", err)
	}

	if time != time2 {
		t.Fatalf("Expected GetGenesisTime to always return same value: %v %v", time, time2)
	}

}

func TestGetGenesisTimeFailure(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)
	defer beaconDB.Close()

	genesisBlock, err := beaconDB.GetCanonicalBlockForSlot(0)
	if err != nil {
		t.Fatalf("Failed to get genesis block: %v", err)
	}

	hash, _ := genesisBlock.Hash()
	err = beaconDB.removeBlock(hash)
	if err != nil {
		t.Fatalf("Failed to remove genesis block: %v", err)
	}

	if _, err := beaconDB.GetGenesisTime(); err == nil {
		t.Fatalf("Expected GetGenesisTime to fail")
	}
}