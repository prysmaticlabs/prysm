package blockchain

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

type faultyDB struct{}

func (f *faultyDB) Get(k []byte) ([]byte, error) {
	return []byte{}, nil
}

func (f *faultyDB) Has(k []byte) (bool, error) {
	return true, nil
}

func (f *faultyDB) Put(k []byte, v []byte) error {
	return nil
}

func (f *faultyDB) Delete(k []byte) error {
	return nil
}

func (f *faultyDB) Close() {}

func (f *faultyDB) NewBatch() ethdb.Batch {
	return nil
}

func startInMemoryBeaconChain(t *testing.T) (*BeaconChain, *database.DB) {
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	beaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	return beaconChain, db
}

func TestNewBeaconChain(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	msg := hook.LastEntry().Message
	want := "No chainstate found on disk, initializing beacon from genesis"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	hook.Reset()
	aState := types.NewGenesisActiveState()
	cState, err := types.NewGenesisCrystallizedState()
	if err != nil {
		t.Errorf("Creating new genesis state failed %v", err)
	}
	if _, err := types.NewGenesisBlock(); err != nil {
		t.Errorf("Creating a new genesis block failed %v", err)
	}

	if !reflect.DeepEqual(beaconChain.ActiveState(), aState) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.ActiveState(), aState)
	}

	if !reflect.DeepEqual(beaconChain.CrystallizedState(), cState) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.CrystallizedState(), cState)
	}
	if _, err := beaconChain.GenesisBlock(); err != nil {
		t.Errorf("Getting new beaconchain genesis failed: %v", err)
	}
}

func TestGetGenesisBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := &pb.BeaconBlock{
		ParentHash: make([]byte, 32),
		Timestamp: &timestamp.Timestamp{
			Seconds: 13000000,
		},
	}
	bytes, err := proto.Marshal(block)
	if err != nil {
		t.Errorf("unable to Marshal genesis block: %v", err)
	}

	if err := db.DB().Put([]byte("genesis"), bytes); err != nil {
		t.Errorf("unable to save key value of genesis: %v", err)
	}

	genesisBlock, err := beaconChain.GenesisBlock()
	if err != nil {
		t.Errorf("unable to get key value of genesis: %v", err)
	}

	time, err := genesisBlock.Timestamp()
	if err != nil {
		t.Errorf("Timestamp could not be retrieved: %v", err)
	}
	if time.Second() != 40 {
		t.Errorf("Timestamp was not saved properly: %v", time.Second())
	}
}

func TestCanonicalHead(t *testing.T) {
	chain, err := NewBeaconChain(&faultyDB{})
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}
	// Using a faultydb that returns true on has, but nil on get should cause
	// proto.Unmarshal to throw error.
	block, err := chain.CanonicalHead()
	if err != nil {
		t.Fatal("expected canonical head to throw error")
	}
	expectedBlock := types.NewBlock(&pb.BeaconBlock{})
	if !reflect.DeepEqual(block, expectedBlock) {
		t.Errorf("mismatched canonical head: expected %v, received %v", expectedBlock, block)
	}
}

func TestSaveCanonicalBlock(t *testing.T) {
	block := types.NewBlock(&pb.BeaconBlock{})
	chain, err := NewBeaconChain(&faultyDB{})
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}
	if err := chain.saveCanonicalBlock(block); err != nil {
		t.Errorf("save canonical should pass: %v", err)
	}
}

func TestSetActiveState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.ActiveState{
		PendingAttestations: []*pb.AttestationRecord{
			{Slot: 0, ShardBlockHash: []byte{1}}, {Slot: 1, ShardBlockHash: []byte{2}},
		},
		RecentBlockHashes: [][]byte{
			{'A'}, {'B'}, {'C'}, {'D'},
		},
	}
	active := types.NewActiveState(data, make(map[[32]byte]*types.VoteCache))

	if err := beaconChain.SetActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.ActiveState, active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconChain.state.ActiveState)
	}

}

func TestSetCrystallizedState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.CrystallizedState{
		CurrentDynasty: 3,
		DynastySeed:    []byte{'A'},
	}
	crystallized := types.NewCrystallizedState(data)

	if err := beaconChain.SetCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.CrystallizedState, crystallized) {
		t.Errorf("crystallized state was not updated. wanted %v, got %v", crystallized, beaconChain.state.CrystallizedState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}

	// The crystallized state should still be the one we mutated and persited earlier.
	if crystallized.CurrentDynasty() != newBeaconChain.state.CrystallizedState.CurrentDynasty() {
		t.Errorf("crystallized state dynasty incorrect. wanted %v, got %v", crystallized.CurrentDynasty(), newBeaconChain.state.CrystallizedState.CurrentDynasty())
	}
	if crystallized.DynastySeed() != newBeaconChain.state.CrystallizedState.DynastySeed() {
		t.Errorf("crystallized state current checkpoint incorrect. wanted %v, got %v", crystallized.DynastySeed(), newBeaconChain.state.CrystallizedState.DynastySeed())
	}
}

func TestSaveAndRemoveBlocks(t *testing.T) {
	b, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Fatalf("unable to generate hash of block %v", err)
	}

	if err := b.saveBlock(block); err != nil {
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

	if err := b.db.Put(key, marshalled); err != nil {
		t.Fatal(err)
	}

	retblock, err := b.getBlock(hash)
	if err != nil {
		t.Fatalf("block is unable to be retrieved")
	}

	if retblock.SlotNumber() != newblock.SlotNumber() {
		t.Errorf("slotnumber does not match for saved and retrieved blocks")
	}

	if !bytes.Equal(retblock.PowChainRef().Bytes(), newblock.PowChainRef().Bytes()) {
		t.Errorf("POW chain ref does not match for saved and retrieved blocks")
	}

	if err := b.removeBlock(hash); err != nil {
		t.Fatalf("error removing block %v", err)
	}

	if _, err := b.getBlock(hash); err == nil {
		t.Fatalf("block is able to be retrieved")
	}

	if err := b.removeBlock(hash); err != nil {
		t.Fatalf("unable to remove block a second time %v", err)
	}
}

func TestCheckBlockBySlotNumber(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	if err := beaconChain.saveCanonicalSlotNumber(block.SlotNumber(), hash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if err := beaconChain.saveBlock(block); err != nil {
		t.Fatalf("unable to save block %v", err)
	}

	slotExists, err := beaconChain.hasCanonicalBlockForSlot(block.SlotNumber())
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

	if err := beaconChain.saveCanonicalSlotNumber(block.SlotNumber(), althash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	retrievedHash, err := beaconChain.db.Get(canonicalBlockKey(block.SlotNumber()))
	if err != nil {
		t.Fatalf("unable to retrieve blockhash %v", err)
	}

	if !bytes.Equal(retrievedHash, althash[:]) {
		t.Errorf("unequal hashes between what was saved and what was retrieved %v, %v", retrievedHash, althash)
	}
}

func TestGetBlockBySlotNumber(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	if err := beaconChain.saveCanonicalSlotNumber(block.SlotNumber(), hash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if err := beaconChain.saveBlock(block); err != nil {
		t.Fatalf("unable to save block %v", err)
	}

	retblock, err := beaconChain.getCanonicalBlockForSlot(block.SlotNumber())
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

	if err := beaconChain.saveCanonicalSlotNumber(block.SlotNumber(), althash); err != nil {
		t.Fatalf("unable to save canonical slot %v", err)
	}

	if _, err = beaconChain.getCanonicalBlockForSlot(block.SlotNumber()); err == nil {
		t.Fatal("there should be an error because block does not exist in the db")
	}
}

func TestSaveAndRemoveAttestations(t *testing.T) {
	b, db := startInMemoryBeaconChain(t)
	defer db.Close()

	attestation := types.NewAttestation(&pb.AttestationRecord{
		Slot:             1,
		ShardId:          1,
		AttesterBitfield: []byte{'A'},
	})

	hash := attestation.Key()
	if err := b.saveAttestation(attestation); err != nil {
		t.Fatalf("unable to save attestation %v", err)
	}

	exist, err := b.hasAttestation(hash)
	if err != nil {
		t.Fatalf("unable to check attestation %v", err)
	}
	if !exist {
		t.Fatal("saved attestation does not exist")
	}

	// Adding a different attestation with the same key
	newAttestation := types.NewAttestation(&pb.AttestationRecord{
		Slot:             2,
		ShardId:          2,
		AttesterBitfield: []byte{'B'},
	})

	key := blockKey(hash)
	marshalled, err := proto.Marshal(newAttestation.Proto())
	if err != nil {
		t.Fatal(err)
	}

	if err := b.db.Put(key, marshalled); err != nil {
		t.Fatal(err)
	}

	returnedAttestation, err := b.getAttestation(hash)
	if err != nil {
		t.Fatalf("attestation is unable to be retrieved")
	}

	if returnedAttestation.SlotNumber() != newAttestation.SlotNumber() {
		t.Errorf("slotnumber does not match for saved and retrieved attestation")
	}

	if !bytes.Equal(returnedAttestation.AttesterBitfield(), newAttestation.AttesterBitfield()) {
		t.Errorf("attester bitfield does not match for saved and retrieved attester")
	}

	if err := b.removeAttestation(hash); err != nil {
		t.Fatalf("error removing attestation %v", err)
	}

	if _, err := b.getAttestation(hash); err == nil {
		t.Fatalf("attestation is able to be retrieved")
	}
}

func TestSaveAndRemoveAttestationHashList(t *testing.T) {
	b, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 0,
	})
	blockHash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	attestation := types.NewAttestation(&pb.AttestationRecord{
		Slot:             1,
		ShardId:          1,
		AttesterBitfield: []byte{'A'},
	})
	attestationHash := attestation.Key()

	if err := b.saveAttestationHash(blockHash, attestationHash); err != nil {
		t.Fatalf("unable to save attestation hash %v", err)
	}

	exist, err := b.hasAttestationHash(blockHash, attestationHash)
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if !exist {
		t.Error("saved attestation hash does not exist")
	}

	// Negative test case: try with random attestation, exist should be false.
	exist, err = b.hasAttestationHash(blockHash, [32]byte{'A'})
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if exist {
		t.Error("attestation hash shouldn't have existed")
	}

	// Remove attestation list by deleting the block hash key.
	if err := b.removeAttestationHashList(blockHash); err != nil {
		t.Fatalf("remove attestation hash list failed %v", err)
	}

	// Negative test case: try with deleted block hash, this should fail.
	_, err = b.hasAttestationHash(blockHash, attestationHash)
	if err == nil {
		t.Error("attestation hash should't have existed in DB")
	}
}
