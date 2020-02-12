package blockchain

import (
	"context"
	"reflect"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_OnAttestation(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = blockTree1(db, []byte{'g'})
	if err != nil {
		t.Fatal(err)
	}

	BlkWithOutState := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 0}}
	if err := db.SaveBlock(ctx, BlkWithOutState); err != nil {
		t.Fatal(err)
	}
	BlkWithOutStateRoot, _ := ssz.HashTreeRoot(BlkWithOutState.Block)

	BlkWithStateBadAtt := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, BlkWithStateBadAtt); err != nil {
		t.Fatal(err)
	}
	BlkWithStateBadAttRoot, _ := ssz.HashTreeRoot(BlkWithStateBadAtt.Block)

	s, err := beaconstate.InitializeFromProto(&pb.BeaconState{})
	if err := s.SetSlot(100 * params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot); err != nil {
		t.Fatal(err)
	}

	BlkWithValidState := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	if err := db.SaveBlock(ctx, BlkWithValidState); err != nil {
		t.Fatal(err)
	}
	BlkWithValidStateRoot, _ := ssz.HashTreeRoot(BlkWithValidState.Block)
	s, _ = stateTrie.InitializeFromProto(&pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err := service.beaconDB.SaveState(ctx, s, BlkWithValidStateRoot); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		a             *ethpb.Attestation
		s             *pb.BeaconState
		wantErr       bool
		wantErrString string
	}{
		{
			name:          "attestation's data slot not aligned with target vote",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "data slot is not in the same epoch as target 1 != 0",
		},
		{
			name:          "attestation's target root not in db",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: []byte{'A'}}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "target root does not exist in db",
		},
		{
			name:          "no pre state for attestations's target block",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "pre state of target block 0 does not exist",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
				Root: BlkWithStateBadAttRoot[:]}}},
			s:             &pb.BeaconState{Slot: 100 * params.BeaconConfig().SlotsPerEpoch},
			wantErr:       true,
			wantErrString: "does not match current epoch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.onAttestation(ctx, tt.a)
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.wantErrString) {
					t.Errorf("Store.onAttestation() error = %v, wantErr = %v", err, tt.wantErrString)
				}
			} else {
				t.Error(err)
			}
		})
	}
}

func TestStore_SaveCheckpointState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)
	params.UseDemoBeaconConfig()

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	s, _ := stateTrie.InitializeFromProto(&pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes:         make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		StateRoots:          make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		BlockRoots:          make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot),
		LatestBlockHeader:   &ethpb.BeaconBlockHeader{},
		JustificationBits:   []byte{0},
		Slashings:           make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	})
	r := [32]byte{'g'}
	if err := service.beaconDB.SaveState(ctx, s, r); err != nil {
		t.Fatal(err)
	}
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}

	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'A'}}
	service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'A'}))
	s1, err := service.getAttPreState(ctx, cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot() != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot())
	}

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: []byte{'B'}}
	service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'B'}))
	s2, err := service.getAttPreState(ctx, cp2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Slot() != 2*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot())
	}

	s1, err = service.getAttPreState(ctx, cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot() != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot())
	}

	s1, err = service.checkpointState.StateByCheckpoint(cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot() != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot())
	}

	s2, err = service.checkpointState.StateByCheckpoint(cp2)
	if err != nil {
		t.Fatal(err)
	}
	if s2.Slot() != 2*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot())
	}

	s.SetSlot(params.BeaconConfig().SlotsPerEpoch + 1)
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	cp3 := &ethpb.Checkpoint{Epoch: 1, Root: []byte{'C'}}
	service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'C'}))
	s3, err := service.getAttPreState(ctx, cp3)
	if err != nil {
		t.Fatal(err)
	}
	if s3.Slot() != s.Slot() {
		t.Errorf("Wanted state slot: %d, got: %d", s.Slot(), s3.Slot())
	}
}

func TestStore_UpdateCheckpointState(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	epoch := uint64(1)
	baseState, _ := testutil.DeterministicGenesisState(t, 1)
	baseState.SetSlot(epoch * params.BeaconConfig().SlotsPerEpoch)
	checkpoint := &ethpb.Checkpoint{Epoch: epoch}
	service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root))
	returned, err := service.getAttPreState(ctx, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if baseState.Slot() != returned.Slot() {
		t.Error("Incorrectly returned base state")
	}

	cached, err := service.checkpointState.StateByCheckpoint(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	if cached == nil {
		t.Error("State should have been cached")
	}

	epoch = uint64(2)
	newCheckpoint := &ethpb.Checkpoint{Epoch: epoch}
	service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root))
	returned, err = service.getAttPreState(ctx, newCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	baseState, err = state.ProcessSlots(ctx, baseState, helpers.StartSlot(newCheckpoint.Epoch))
	if err != nil {
		t.Fatal(err)
	}
	if baseState.Slot() != returned.Slot() {
		t.Error("Incorrectly returned base state")
	}

	cached, err = service.checkpointState.StateByCheckpoint(newCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(returned, cached) {
		t.Error("Incorrectly cached base state")
	}
}

func TestAttEpoch_MatchPrevEpoch(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.verifyAttTargetEpoch(
		ctx,
		0,
		params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot,
		&ethpb.Checkpoint{}); err != nil {
		t.Error(err)
	}
}

func TestAttEpoch_MatchCurrentEpoch(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.verifyAttTargetEpoch(
		ctx,
		0,
		params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot,
		&ethpb.Checkpoint{Epoch: 1}); err != nil {
		t.Error(err)
	}
}

func TestAttEpoch_NotMatch(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.verifyAttTargetEpoch(
		ctx,
		0,
		2*params.BeaconConfig().SlotsPerEpoch*params.BeaconConfig().SecondsPerSlot,
		&ethpb.Checkpoint{}); !strings.Contains(err.Error(),
		"target epoch 0 does not match current epoch 2 or prev epoch 1") {
		t.Error("Did not receive wanted error")
	}
}

func TestVerifyBeaconBlock_NoBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	d := &ethpb.AttestationData{}
	if err := service.verifyBeaconBlock(ctx, d); !strings.Contains(err.Error(), "beacon block  does not exist") {
		t.Error("Did not receive the wanted error")
	}
}

func TestVerifyBeaconBlock_futureBlock(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	service.beaconDB.SaveBlock(ctx, b)
	r, _ := ssz.HashTreeRoot(b.Block)
	d := &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: r[:]}

	if err := service.verifyBeaconBlock(ctx, d); !strings.Contains(err.Error(), "could not process attestation for future block") {
		t.Error("Did not receive the wanted error")
	}
}

func TestVerifyBeaconBlock_OK(t *testing.T) {
	ctx := context.Background()
	db := testDB.SetupDB(t)
	defer testDB.TeardownDB(t, db)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	service.beaconDB.SaveBlock(ctx, b)
	r, _ := ssz.HashTreeRoot(b.Block)
	d := &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: r[:]}

	if err := service.verifyBeaconBlock(ctx, d); err != nil {
		t.Error("Did not receive the wanted error")
	}
}
