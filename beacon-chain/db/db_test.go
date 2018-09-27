package db

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func startInMemoryBeaconDB(t *testing.T) *BeaconDB {
	config := Config{Path: "", Name: "", InMemory: true}
	db, err := NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}

	return db
}

func TestNewDB(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := startInMemoryBeaconDB(t)

	msg := hook.LastEntry().Message
	want := "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	aState := types.NewGenesisActiveState()
	cState, err := types.NewGenesisCrystallizedState("")
	if err != nil {
		t.Errorf("Creating new genesis state failed %v", err)
	}

	if !proto.Equal(beaconDB.GetActiveState().Proto(), aState.Proto()) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconDB.GetActiveState(), aState)
	}

	if !proto.Equal(beaconDB.GetCrystallizedState().Proto(), cState.Proto()) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconDB.GetCrystallizedState(), cState)
	}
}

func TestSetActiveState(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)

	data := &pb.ActiveState{
		PendingAttestations: []*pb.AggregatedAttestation{
			{Slot: 0, ShardBlockHash: []byte{1}}, {Slot: 1, ShardBlockHash: []byte{2}},
		},
		RecentBlockHashes: [][]byte{
			{'A'}, {'B'}, {'C'}, {'D'},
		},
	}
	active := types.NewActiveState(data, make(map[[32]byte]*types.VoteCache))

	if err := beaconDB.SaveActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconDB.state.aState, active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconDB.state.aState)
	}

}

func TestSetCrystallizedState(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)

	data := &pb.CrystallizedState{
		CurrentDynasty: 3,
		DynastySeed:    []byte{'A'},
	}
	crystallized := types.NewCrystallizedState(data)

	if err := beaconDB.SaveCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if !reflect.DeepEqual(beaconDB.state.cState, crystallized) {
		t.Errorf("crystallized state was not updated. wanted %v, got %v", crystallized, beaconDB.state.cState)
	}
}

func TestSaveAndRemoveBlocks(t *testing.T) {
	beaconDB := startInMemoryBeaconDB(t)

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
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
		SlotNumber:  4,
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

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
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
		SlotNumber:  64,
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

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
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
		SlotNumber:  64,
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
