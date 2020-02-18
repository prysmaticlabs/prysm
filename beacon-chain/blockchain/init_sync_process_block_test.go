package blockchain

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"gopkg.in/d4l3k/messagediff.v1"
)

func TestFilterBoundaryCandidates_FilterCorrect(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})

	for i := uint64(0); i < 500; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
		if helpers.IsEpochStart(i) {
			service.boundaryRoots = append(service.boundaryRoots, root)
		}
	}
	lastIndex := len(service.boundaryRoots) - 1
	for i := uint64(500); i < 2000; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
	}
	// Set current state.
	latestSlot := helpers.RoundUpToNearestEpoch(2000)
	st.SetSlot(latestSlot)
	lastRoot := [32]byte{}
	copy(lastRoot[:], bytesutil.Bytes32(latestSlot))

	service.initSyncState[lastRoot] = st.Copy()
	service.finalizedCheckpt = &ethpb.Checkpoint{
		Epoch: 0,
		Root:  []byte{},
	}
	service.filterBoundaryCandidates(context.Background(), lastRoot, st)
	if len(service.boundaryRoots[lastIndex+1:]) == 0 {
		t.Fatal("Wanted non zero added boundary roots")
	}
	for _, rt := range service.boundaryRoots[lastIndex+1:] {
		st, ok := service.initSyncState[rt]
		if !ok {
			t.Error("Root doen't exist in cache map")
			continue
		}
		if !(helpers.IsEpochStart(st.Slot()) || helpers.IsEpochStart(st.Slot()-1) || helpers.IsEpochStart(st.Slot()+1)) {
			t.Errorf("boundary roots not validly stored. They have slot %d", st.Slot())
		}
	}
}

func TestFilterBoundaryCandidates_HandleSkippedSlots(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})

	for i := uint64(0); i < 500; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
		if helpers.IsEpochStart(i) {
			service.boundaryRoots = append(service.boundaryRoots, root)
		}
	}
	lastIndex := len(service.boundaryRoots) - 1
	for i := uint64(500); i < 2000; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		// save only for offsetted slots
		if helpers.IsEpochStart(i + 10) {
			service.initSyncState[root] = st.Copy()
		}
	}
	// Set current state.
	latestSlot := helpers.RoundUpToNearestEpoch(2000)
	st.SetSlot(latestSlot)
	lastRoot := [32]byte{}
	copy(lastRoot[:], bytesutil.Bytes32(latestSlot))

	service.initSyncState[lastRoot] = st.Copy()
	service.finalizedCheckpt = &ethpb.Checkpoint{
		Epoch: 0,
		Root:  []byte{},
	}
	service.filterBoundaryCandidates(context.Background(), lastRoot, st)
	if len(service.boundaryRoots[lastIndex+1:]) == 0 {
		t.Fatal("Wanted non zero added boundary roots")
	}
	for _, rt := range service.boundaryRoots[lastIndex+1:] {
		st, ok := service.initSyncState[rt]
		if !ok {
			t.Error("Root doen't exist in cache map")
			continue
		}
		if st.Slot() >= 500 {
			// Ignore head boundary root.
			if st.Slot() == 2016 {
				continue
			}
			if !helpers.IsEpochStart(st.Slot() + 10) {
				t.Errorf("boundary roots not validly stored. They have slot %d "+
					"instead of the offset from epoch start", st.Slot())
			}
		}
	}
}

func TestPruneOldStates_AlreadyFinalized(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})

	for i := uint64(100); i < 200; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
		service.boundaryRoots = append(service.boundaryRoots, root)
	}
	finalizedEpoch := uint64(5)
	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: finalizedEpoch}
	service.pruneOldStates()
	for _, rt := range service.boundaryRoots {
		st, ok := service.initSyncState[rt]
		if !ok {
			t.Error("Root doen't exist in cache map")
			continue
		}
		if st.Slot() < helpers.StartSlot(finalizedEpoch) {
			t.Errorf("State with slot %d still exists and not pruned", st.Slot())
		}
	}
}

func TestPruneNonBoundary_CanPrune(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := stateTrie.InitializeFromProtoUnsafe(&pb.BeaconState{})

	for i := uint64(0); i < 2000; i++ {
		st.SetSlot(i)
		root := [32]byte{}
		copy(root[:], bytesutil.Bytes32(i))
		service.initSyncState[root] = st.Copy()
		if helpers.IsEpochStart(i) {
			service.boundaryRoots = append(service.boundaryRoots, root)
		}
	}
	service.pruneNonBoundaryStates()
	for _, rt := range service.boundaryRoots {
		st, ok := service.initSyncState[rt]
		if !ok {
			t.Error("Root doesn't exist in cache map")
			continue
		}
		if !helpers.IsEpochStart(st.Slot()) {
			t.Errorf("Non boundary state with slot %d still exists and not pruned", st.Slot())
		}
	}
}

func TestGenerateState_CorrectlyGenerated(t *testing.T) {
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	cfg := &Config{BeaconDB: db}
	service, err := NewService(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	beaconState, privs := testutil.DeterministicGenesisState(t, 32)
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock.Block)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.SetLatestBlockHeader(&ethpb.BeaconBlockHeader{
		Slot:       genesisBlock.Block.Slot,
		ParentRoot: genesisBlock.Block.ParentRoot,
		StateRoot:  params.BeaconConfig().ZeroHash[:],
		BodyRoot:   bodyRoot[:],
	})
	beaconState.SetSlashings(make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector))
	cp := beaconState.CurrentJustifiedCheckpoint()
	mockRoot := [32]byte{}
	copy(mockRoot[:], "hello-world")
	cp.Root = mockRoot[:]
	beaconState.SetCurrentJustifiedCheckpoint(cp)
	beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{})
	err = db.SaveBlock(context.Background(), genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	genRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	err = db.SaveState(context.Background(), beaconState, genRoot)
	if err != nil {
		t.Fatal(err)
	}

	lastBlock := &ethpb.SignedBeaconBlock{}
	for i := uint64(1); i < 10; i++ {
		block, err := testutil.GenerateFullBlock(beaconState, privs, testutil.DefaultBlockGenConfig(), i)
		if err != nil {
			t.Fatal(err)
		}
		beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block)
		if err != nil {
			t.Fatal(err)
		}
		err = db.SaveBlock(context.Background(), block)
		if err != nil {
			t.Fatal(err)
		}
		lastBlock = block
	}
	root, err := ssz.HashTreeRoot(lastBlock.Block)
	if err != nil {
		t.Fatal(err)
	}

	newState, err := service.generateState(context.Background(), genRoot, root)
	if err != nil {
		t.Fatal(err)
	}
	if !ssz.DeepEqual(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe()) {
		diff, _ := messagediff.PrettyDiff(newState.InnerStateUnsafe(), beaconState.InnerStateUnsafe())
		t.Errorf("Generated state is different from what is expected: %s", diff)
	}
}
