package blockchain

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
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
	active, crystallized := types.NewGenesisStates()
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
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if !canProcess {
		t.Error("Should be able to process block, could not")
	}

	// Negative scenario #1, invalid active hash
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       []byte{'A'},
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid active hash")
	}

	// Negative scenario #2, invalid crystallized hash
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: []byte{'A'},
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid crystallied hash")
	}

	// Negative scenario #3, invalid timestamp
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

func TestProcessBlockWithBadHashes(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	// Test negative scenario where active state hash is different than node's compute.
	parentBlock := NewBlock(t, nil)
	parentHash, err := parentBlock.Hash()
	if err != nil {
		t.Fatalf("Failed to compute parent block's hash: %v", err)
	}
	if err = db.DB().Put(parentHash[:], []byte{}); err != nil {
		t.Fatalf("Failed to put parent block on db: %v", err)
	}

	// Initialize state.
	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}}})
	activeStateHash, err := active.Hash()
	if err != nil {
		t.Fatalf("Cannot hash active state: %v", err)
	}

	crystallized := types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 10000})
	crystallizedStateHash, err := crystallized.Hash()
	if err != nil {
		t.Fatalf("Cannot hash crystallized state: %v", err)
	}

	block := NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            1,
		ActiveStateHash:       activeStateHash[:],
		CrystallizedStateHash: crystallizedStateHash[:],
		ParentHash:            parentHash[:],
	})

	// Test negative scenario where active state hash is different than node's compute.
	beaconChain.state.ActiveState = types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'B'}}})

	canProcess, err := beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("CanProcessBlocks should have returned false with diff state hashes")
	}

	// Test negative scenario where crystallized state hash is different than node's compute.
	beaconChain.state.CrystallizedState = types.NewCrystallizedState(&pb.CrystallizedState{LastStateRecalc: 9999})

	canProcess, err = beaconChain.CanProcessBlock(&mockFetcher{}, block, true)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("CanProcessBlocks should have returned false with diff state hashes")
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

func TestClearRecentBlockHashes(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	active := types.NewActiveState(&pb.ActiveState{RecentBlockHashes: [][]byte{{'A'}, {'B'}, {'C'}}})
	if err := beaconChain.SetActiveState(active); err != nil {
		t.Fatalf("unable to Mutate Active state: %v", err)
	}
	if reflect.DeepEqual(beaconChain.state.ActiveState.RecentBlockHashes(), [][]byte{{'A'}, {'B'}, {'C'}}) {
		t.Fatalf("recent block hash was not saved: %d", beaconChain.state.ActiveState.RecentBlockHashes())
	}
	beaconChain.ActiveState().ClearRecentBlockHashes()
	if reflect.DeepEqual(beaconChain.state.ActiveState.RecentBlockHashes(), [][]byte{}) {
		t.Fatalf("attester deposit was not able to be reset: %d", beaconChain.state.ActiveState.RecentBlockHashes())
	}
}

func TestUpdateJustifiedSlot(t *testing.T) {
	beaconChain, db := startInMemoryBeaconChain(t)
	defer db.Close()

	data := &pb.CrystallizedState{LastStateRecalc: 5 * params.CycleLength, LastJustifiedSlot: 4, LastFinalizedSlot: 3}
	beaconChain.SetCrystallizedState(types.NewCrystallizedState(data))
	if beaconChain.state.CrystallizedState.LastFinalizedSlot() != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedSlot() != uint64(4) {
		t.Fatal("crystallized state unable to be saved")
	}

	beaconChain.state.CrystallizedState.UpdateJustifiedSlot(5)

	if beaconChain.state.CrystallizedState.LastJustifiedSlot() != uint64(5) {
		t.Fatalf("unable to update last justified Slot: %d", beaconChain.state.CrystallizedState.LastJustifiedSlot())
	}
	if beaconChain.state.CrystallizedState.LastFinalizedSlot() != uint64(4) {
		t.Fatalf("unable to update last finalized Slot: %d", beaconChain.state.CrystallizedState.LastFinalizedSlot())
	}

	data = &pb.CrystallizedState{LastStateRecalc: 8 * params.CycleLength, LastJustifiedSlot: 4, LastFinalizedSlot: 3}
	beaconChain.SetCrystallizedState(types.NewCrystallizedState(data))

	if beaconChain.state.CrystallizedState.LastFinalizedSlot() != uint64(3) ||
		beaconChain.state.CrystallizedState.LastJustifiedSlot() != uint64(4) {
		t.Fatal("crystallized state unable to be saved")
	}

	beaconChain.state.CrystallizedState.UpdateJustifiedSlot(8)

	if beaconChain.state.CrystallizedState.LastJustifiedSlot() != uint64(8) {
		t.Fatalf("unable to update last justified Slot: %d", beaconChain.state.CrystallizedState.LastJustifiedSlot())
	}
	if beaconChain.state.CrystallizedState.LastFinalizedSlot() != uint64(3) {
		t.Fatalf("unable to update last finalized Slot: %d", beaconChain.state.CrystallizedState.LastFinalizedSlot())
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

	// Negative scenario #1, invalid crystallized state hash
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       activeHash[:],
		CrystallizedStateHash: []byte{'A'},
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(nil, block, false)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid crystallized hash")
	}

	// Negative scenario #2, invalid active sate hash
	block = NewBlock(t, &pb.BeaconBlock{
		SlotNumber:            2,
		ActiveStateHash:       []byte{'A'},
		CrystallizedStateHash: crystallizedHash[:],
		ParentHash:            parentHash[:],
	})
	canProcess, err = beaconChain.CanProcessBlock(nil, block, false)
	if err == nil {
		t.Fatalf("CanProcessBlocks failed: %v", err)
	}
	if canProcess {
		t.Error("Should not be able to process block with invalid active hash")
	}

	// Negative scenario #3, invalid timestamp
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
