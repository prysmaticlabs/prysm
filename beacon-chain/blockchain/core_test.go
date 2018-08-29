package blockchain

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/prysmaticlabs/prysm/beacon-chain/params"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// FakeClock represents an mocked clock for testing purposes.
type fakeClock struct{}

// Now represents the mocked functionality of a Clock.Now().
func (fakeClock) Now() time.Time {
	return time.Date(1970, 2, 1, 1, 0, 0, 0, time.UTC)
}

type faultyFetcher struct{}

func (f *faultyFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	return nil, errors.New("cannot fetch block")
}

type mockFetcher struct{}

func (m *mockFetcher) BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error) {
	block := gethTypes.NewBlock(&gethTypes.Header{}, nil, nil, nil)
	return block, nil
}

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
	active, crystallized, err := types.NewGenesisStates()
	if err != nil {
		t.Errorf("Creating new genesis state failed %v", err)
	}
	if _, err := types.NewGenesisBlock(); err != nil {
		t.Errorf("Creating a new genesis block failed %v", err)
	}
	if !reflect.DeepEqual(beaconChain.ActiveState(), active) {
		t.Errorf("active states not equal. received: %v, wanted: %v", beaconChain.ActiveState(), active)
	}
	if !reflect.DeepEqual(beaconChain.CrystallizedState(), crystallized) {
		t.Errorf("crystallized states not equal. received: %v, wanted: %v", beaconChain.CrystallizedState(), crystallized)
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

func TestSaveCanonical(t *testing.T) {
	block := types.NewBlock(&pb.BeaconBlock{})
	chain, err := NewBeaconChain(&faultyDB{})
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}
	if err := chain.saveCanonical(block); err != nil {
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
	active := types.NewActiveState(data)

	if err := beaconChain.SetActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}
	if !reflect.DeepEqual(beaconChain.state.ActiveState, active) {
		t.Errorf("active state was not updated. wanted %v, got %v", active, beaconChain.state.ActiveState)
	}

	// Initializing a new beacon chain should deserialize persisted state from disk.
	newBeaconChain, err := NewBeaconChain(db.DB())
	if err != nil {
		t.Fatalf("unable to setup second beacon chain: %v", err)
	}

	// The active state should still be the one we mutated and persited earlier
	for i, hash := range active.RecentBlockHashes() {
		if hash.Hex() != newBeaconChain.ActiveState().RecentBlockHashes()[i].Hex() {
			t.Errorf("active state block hash. wanted %v, got %v", hash.Hex(), newBeaconChain.ActiveState().RecentBlockHashes()[i].Hex())
		}
	}
	if reflect.DeepEqual(active.PendingAttestations(), newBeaconChain.state.ActiveState.RecentBlockHashes()) {
		t.Errorf("active state pending attestation incorrect. wanted %v, got %v", active.PendingAttestations(), newBeaconChain.state.ActiveState.RecentBlockHashes())
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

func TestCanProcessBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	clock = &fakeClock{}

	// Initialize a parent block.
	parentBlock := NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 1,
	})
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	// Using a faulty fetcher should throw an error.
	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 2,
	})
	if _, err := beaconChain.CanProcessBlock(&faultyFetcher{}, block, true); err == nil {
		t.Error("Using a faulty fetcher should throw an error, received nil")
	}

	// Initialize initial state.
	activeState := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}})
	beaconChain.state.ActiveState = activeState
	activeHash, err := activeState.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{})
	beaconChain.state.CrystallizedState = crystallized
	crystallizedHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Compute crystallized state hash failed: %v", err)
	}

	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})

	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err != nil {
		t.Fatalf("canProcessBlocks failed: %v", err)
	}
	if !canProcess {
		t.Error("Should be able to process block, could not")
	}

	// Negative scenario, invalid timestamp
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            1000000,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})

	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid timestamp condition")
	}
}

func TestIsSlotTransition(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	if err := beaconChain.SetCrystallizedState(types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: params.CycleLength})); err != nil {
		t.Fatalf("unable to mutate crystallizedstate: %v", err)
	}
	if !beaconChain.IsCycleTransition(128) {
		t.Errorf("there was supposed to be a slot transition but there isn't one now")
	}
	if beaconChain.IsCycleTransition(80) {
		t.Errorf("there is not supposed to be a slot transition but there is one now")
	}
}

func TestCanProcessBlockObserver(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	clock = &fakeClock{}

	// Initialize a parent block.
	parentBlock := NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 1,
	})
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	// Initialize initial state.
	activeState := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}})
	beaconChain.state.ActiveState = activeState
	activeHash, err := activeState.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{})
	beaconChain.state.CrystallizedState = crystallized
	crystallizedHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Compute crystallized state hash failed: %v", err)
	}

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})

	// A properly initialize block should not fail.
	canProcess, err := beaconChain.CanProcessBlock(nil, block, false)
	if err != nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if !canProcess {
		t.Error("Should be able to process block, could not")
	}

	// Negative scenario, invalid timestamp
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            1000000,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(nil, block, false)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid timestamp condition")
	}
}

func TestSaveBlockWithNil(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	if err := beaconChain.saveBlock(&types.Block{}); err == nil {
		t.Error("Save block should have failed with nil block")
	}
}

func TestVerifyActiveHashWithNil(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()
	beaconChain.SetActiveState(&types.ActiveState{})
	_, err := beaconChain.verifyBlockActiveHash(&types.Block{})
	if err == nil {
		t.Error("Verify block hash should have failed with nil active state")
	}
}

func TestVerifyCrystallizedHashWithNil(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()
	beaconChain.SetCrystallizedState(&types.CrystallizedState{})
	_, err := beaconChain.verifyBlockCrystallizedHash(&types.Block{})
	if err == nil {
		t.Error("Verify block hash should have failed with nil crystallized")
	}
}

func TestComputeCrystallizedState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()
	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 0})
	beaconChain.SetCrystallizedState(crystallized)
	data := &pb.ActiveState{
		PendingAttestations: []*pb.AttestationRecord{
			{Slot: 0, ShardBlockHash: []byte{1}}, {Slot: 1, ShardBlockHash: []byte{2}},
		},
		RecentBlockHashes: [][]byte{
			{'A'}, {'B'}, {'C'}, {'D'},
		},
	}
	active := types.NewActiveState(data)
	block := types.NewBlock(&pb.BeaconBlock{SlotNumber: 1})
	if _, err := beaconChain.computeNewCrystallizedState(active, block); err != nil {
		t.Errorf("computing crystallized state should not have failed: %v", err)
	}
}

func TestComputeActiveState(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()
	_, crystallized, err := types.NewGenesisStates()
	if err != nil {
		t.Fatalf("Can't generate genesis state: %v", err)
	}
	beaconChain.SetCrystallizedState(crystallized)
	if _, err := beaconChain.computeNewActiveState(common.BytesToHash([]byte("chain"))); err != nil {
		t.Errorf("computing active state should not have failed: %v", err)
	}
}

func TestCanProcessAttestations(t *testing.T) {
	bc, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// Process attestation on this block should fail because AttestationRecord's slot # > than block's slot #.
	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 1,
		Attestations: []*pb.AttestationRecord{
			{Slot: 2, ShardId: 0},
		},
	})
	if err := bc.processAttestations(block); err == nil {
		t.Error("Process attestation should have failed because attestation slot # > block #")
	}

	// Process attestation on this should fail because AttestationRecord's slot # > than block's slot #.
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 2 + params.CycleLength,
		Attestations: []*pb.AttestationRecord{
			{Slot: 1, ShardId: 0},
		},
	})
	if err := bc.processAttestations(block); err == nil {
		t.Error("Process attestation should have failed because attestation slot # < block # + cycle length")
	}

	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 0,
		Attestations: []*pb.AttestationRecord{
			{Slot: 0, ShardId: 0, ObliqueParentHashes: [][]byte{{'A'}, {'B'}, {'C'}}},
		},
	})
	var recentBlockHashes [][]byte
	for i := 0; i < params.CycleLength; i++ {
		recentBlockHashes = append(recentBlockHashes, []byte{'X'})
	}
	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: recentBlockHashes})
	if err := bc.SetActiveState(active); err != nil {
		t.Fatalf("unable to mutate active state: %v", err)
	}

	// Process attestation on this crystallized state should fail because only committee is in shard 1.
	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{
		LastStateRecalc: 64,
		IndicesForSlots: []*pb.ShardAndCommitteeArray{
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{ShardId: 1, Committee: []uint32{0, 1, 2, 3, 4, 5}},
				},
			},
		},
	})
	if err := bc.SetCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}
	if err := bc.processAttestations(block); err == nil {
		t.Error("Process attestation should have failed, there's no committee in shard 0")
	}

	// Process attestation should work now, there's a committee in shard 0.
	crystallized = types.NewCrystallizedState(&pb.CrystallizedState{
		LastStateRecalc: 64,
		IndicesForSlots: []*pb.ShardAndCommitteeArray{
			{
				ArrayShardAndCommittee: []*pb.ShardAndCommittee{
					{ShardId: 0, Committee: []uint32{0, 1, 2, 3, 4, 5}},
				},
			},
		},
	})
	if err := bc.SetCrystallizedState(crystallized); err != nil {
		t.Fatalf("unable to mutate crystallized state: %v", err)
	}

	// Process attestation should fail because attester bit field has incorrect length.
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 0,
		Attestations: []*pb.AttestationRecord{
			{Slot: 0, ShardId: 0, AttesterBitfield: []byte{'A', 'B', 'C'}},
		},
	})
	if err := bc.processAttestations(block); err == nil {
		t.Error("Process attestation should have failed, incorrect attester bit field length")
	}

	// Set attester bitfield to the right length.
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 0,
		Attestations: []*pb.AttestationRecord{
			{Slot: 0, ShardId: 0, AttesterBitfield: []byte{'a'}},
		},
	})
	// Process attestation should fail because the non-zero leading bits for votes.
	// a is 01100001
	if err := bc.processAttestations(block); err == nil {
		t.Error("Process attestation should have failed, incorrect attester bit field length")
	}

	// Process attestation should work with correct bitfield bits.
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber: 0,
		Attestations: []*pb.AttestationRecord{
			{Slot: 0, ShardId: 0, AttesterBitfield: []byte{'0'}},
		},
	})
	if err := bc.processAttestations(block); err != nil {
		t.Error(err)
	}
}

func TestSaveProcessedBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	key := blockKey(block.SlotNumber(), hash)

	if err := beaconChain.saveProcessedBlockToDB(block); err != nil {
		t.Errorf("Could not save processed block %v", err)
	}

	enc, err := beaconChain.db.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	unmarshalled := &pb.BeaconBlock{}

	if err := proto.Unmarshal(enc, unmarshalled); err != nil {
		t.Fatal(err)
	}

	retrievedblock := NewBlock(t, unmarshalled)

	if !bytes.Equal(block.PowChainRef().Bytes(), retrievedblock.PowChainRef().Bytes()) {
		t.Fatal("block retrieved does not have the same POW chain ref as the block saved")
	}

}

func TestSaveProcessedBlockWithBlockRegistry(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	registryKey := blockRegistryKey(block.SlotNumber())
	blockhashes := make([][]byte, 0)
	blockhashes = append(blockhashes, []byte{'a'})

	registry := &pb.BlockRegistry{Blockhashes: blockhashes}
	marshalled, err := proto.Marshal(registry)

	if err := beaconChain.db.Put(registryKey, marshalled); err != nil {
		t.Fatal(err)
	}

	key := blockKey(block.SlotNumber(), hash)

	if err := beaconChain.saveProcessedBlockToDB(block); err != nil {
		t.Errorf("Could not save processed block %v", err)
	}

	enc, err := beaconChain.db.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	unmarshalled := &pb.BeaconBlock{}

	if err := proto.Unmarshal(enc, unmarshalled); err != nil {
		t.Fatal(err)
	}

	retrievedblock := NewBlock(t, unmarshalled)

	if !bytes.Equal(block.PowChainRef().Bytes(), retrievedblock.PowChainRef().Bytes()) {
		t.Fatal("block retrieved does not have the same POW chain ref as the block saved")
	}

	enc, err = beaconChain.db.Get(registryKey)
	if err != nil {
		t.Fatal(err)
	}

	unmarshalledRegistry := &pb.BlockRegistry{}
	if err := proto.Unmarshal(enc, unmarshalledRegistry); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(unmarshalledRegistry.GetBlockhashes()[0], []byte{'a'}) {
		t.Errorf("blockhashes are unequal to each other %v , %v", unmarshalledRegistry.GetBlockhashes()[0], []byte{'a'})
	}

	if !bytes.Equal(unmarshalledRegistry.GetBlockhashes()[1], hash[:]) {
		t.Errorf("incorrect blockhash saved %v", hash)
	}

}

func TestHashRegistered(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	registryKey := blockRegistryKey(block.SlotNumber())
	blockhashes := make([][]byte, 0)
	blockhashes = append(blockhashes, hash[:])

	registry := &pb.BlockRegistry{Blockhashes: blockhashes}
	marshalled, err := proto.Marshal(registry)

	if err := beaconChain.db.Put(registryKey, marshalled); err != nil {
		t.Fatal(err)
	}

	if err := beaconChain.saveProcessedBlockToDB(block); err != nil {
		t.Errorf("Could not save processed block %v", err)
	}

	enc, err := beaconChain.db.Get(registryKey)
	if err != nil {
		t.Fatal(err)
	}

	unmarshalledRegistry := &pb.BlockRegistry{}
	if err := proto.Unmarshal(enc, unmarshalledRegistry); err != nil {
		t.Fatal(err)
	}

	if len(unmarshalledRegistry.Blockhashes) > 1 {
		t.Fatal("same hash is being saved twice in the db")
	}

}

func TestRetrieveBlock(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	key := blockKey(block.SlotNumber(), hash)
	marshalled, err := proto.Marshal(block.Proto())
	if err != nil {
		t.Fatal(err)
	}

	if err := beaconChain.db.Put(key, marshalled); err != nil {
		t.Fatal(err)
	}

	retBlock, err := beaconChain.retrieveBlock(block.SlotNumber(), hash)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(block.PowChainRef().Bytes(), retBlock.PowChainRef().Bytes()) {
		t.Fatal("block retrieved does not have the same POW chain ref as the block saved")
	}

}

func TestRetrieveBlockRegistry(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:  64,
		PowChainRef: []byte("a"),
	})

	hash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	registryKey := blockRegistryKey(block.SlotNumber())
	blockhashes := make([][]byte, 0)
	blockhashes = append(blockhashes, hash[:])

	registry := &pb.BlockRegistry{Blockhashes: blockhashes}
	marshalled, err := proto.Marshal(registry)

	if err := beaconChain.db.Put(registryKey, marshalled); err != nil {
		t.Fatal(err)
	}

	retRegistry, err := beaconChain.retrieveBlockRegistry(block.SlotNumber())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(hash[:], retRegistry[0]) {
		t.Fatal("blockhash saved is not the same as blockhash retrieved")
	}

}

// NewBlock is a helper method to create blocks with valid defaults.
// For a generic block, use NewBlock(t, nil).
func NewBlock(t *testing.T, b *pb.BeaconBlock) *types.Block {
	if b == nil {
		b = &pb.BeaconBlock{}
	}
	if b.ActiveStateHash == nil {
		b.ActiveStateHash = make([]byte, 32)
	}
	if b.CrystallizedStateHash == nil {
		b.CrystallizedStateHash = make([]byte, 32)
	}
	if b.ParentHash == nil {
		b.ParentHash = make([]byte, 32)
	}

	return types.NewBlock(b)
}
