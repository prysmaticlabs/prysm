package blockchain

import (
	"context"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestStore_OnAttestation(t *testing.T) {
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB:        db,
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		StateGen:        stategen.New(db, sc),
	}
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
	BlkWithOutStateRoot, err := stateutil.BlockRoot(BlkWithOutState.Block)
	if err != nil {
		t.Fatal(err)
	}

	BlkWithStateBadAtt := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	if err := db.SaveBlock(ctx, BlkWithStateBadAtt); err != nil {
		t.Fatal(err)
	}
	BlkWithStateBadAttRoot, err := stateutil.BlockRoot(BlkWithStateBadAtt.Block)
	if err != nil {
		t.Fatal(err)
	}

	s := testutil.NewBeaconState()
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

	BlkWithValidStateRoot, err := stateutil.BlockRoot(BlkWithValidState.Block)
	if err != nil {
		t.Fatal(err)
	}
	s = testutil.NewBeaconState()
	if err := s.SetFork(&pb.Fork{
		Epoch:           0,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
	}); err != nil {
		t.Fatal(err)
	}
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
			wantErrString: "could not get pre state for slot 0",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
				Root: BlkWithStateBadAttRoot[:]}}},
			s:             &pb.BeaconState{Slot: 100 * params.BeaconConfig().SlotsPerEpoch},
			wantErr:       true,
			wantErrString: "target epoch 100 does not match current epoch",
		},
		{
			name:          "process nil field (a.Target) in attestation",
			a:             nil,
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "nil attestation",
		},
		{
			name:          "process nil field (a.Data) in attestation",
			a:             &ethpb.Attestation{},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "nil attestation.Data field",
		},
		{
			name:          "process nil field (a.Target) in attestation",
			a:             &ethpb.Attestation{Data: &ethpb.AttestationData{}},
			s:             &pb.BeaconState{},
			wantErr:       true,
			wantErrString: "nil attestation.Data.Target field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.onAttestation(ctx, tt.a)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrString) {
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
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	s, err := stateTrie.InitializeFromProto(&pb.BeaconState{
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
		FinalizedCheckpoint: &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'A'}, 32)},
		Validators:          []*ethpb.Validator{{PublicKey: bytesutil.PadTo([]byte("foo"), 48)}},
		Balances:            []uint64{0},
	})
	if err != nil {
		t.Fatal(err)
	}
	r := [32]byte{'g'}
	if err := service.beaconDB.SaveState(ctx, s, r); err != nil {
		t.Fatal(err)
	}

	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}

	r = bytesutil.ToBytes32([]byte{'A'})
	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, 32)}
	if err := service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'A'})); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'A'}, 32)}); err != nil {
		t.Fatal(err)
	}

	s1, err := service.getAttPreState(ctx, cp1)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Slot() != 1*params.BeaconConfig().SlotsPerEpoch {
		t.Errorf("Wanted state slot: %d, got: %d", 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot())
	}

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}
	if err := service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'B'})); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'B'}, 32)}); err != nil {
		t.Fatal(err)
	}
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

	if err := s.SetSlot(params.BeaconConfig().SlotsPerEpoch + 1); err != nil {
		t.Fatal(err)
	}
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	cp3 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, 32)}
	if err := service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'C'})); err != nil {
		t.Fatal(err)
	}
	if err := service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'C'}, 32)}); err != nil {
		t.Fatal(err)
	}
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
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	epoch := uint64(1)
	baseState, _ := testutil.DeterministicGenesisState(t, 1)
	if err := baseState.SetSlot(epoch * params.BeaconConfig().SlotsPerEpoch); err != nil {
		t.Fatal(err)
	}
	checkpoint := &ethpb.Checkpoint{Epoch: epoch}
	if err := service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root)); err != nil {
		t.Fatal(err)
	}
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
	if err := service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root)); err != nil {
		t.Fatal(err)
	}
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
	if !proto.Equal(returned.InnerStateUnsafe(), cached.InnerStateUnsafe()) {
		t.Error("Incorrectly cached base state")
	}
}

func TestAttEpoch_MatchPrevEpoch(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

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
	db, _ := testDB.SetupDB(t)

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
	db, _ := testDB.SetupDB(t)

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
	db, _ := testDB.SetupDB(t)

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
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	r, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	d := &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: r[:]}

	err = service.verifyBeaconBlock(ctx, d)
	if err == nil || !strings.Contains(err.Error(), "could not process attestation for future block") {
		t.Error("Did not receive the wanted error")
	}
}

func TestVerifyBeaconBlock_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	if err := service.beaconDB.SaveBlock(ctx, b); err != nil {
		t.Fatal(err)
	}
	r, err := stateutil.BlockRoot(b.Block)
	if err != nil {
		t.Fatal(err)
	}
	d := &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: r[:]}

	if err := service.verifyBeaconBlock(ctx, d); err != nil {
		t.Error("Did not receive the wanted error")
	}
}

func TestVerifyLMDFFGConsistent_NotOK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b32 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 32}}
	if err := service.beaconDB.SaveBlock(ctx, b32); err != nil {
		t.Fatal(err)
	}
	r32, err := stateutil.BlockRoot(b32.Block)
	if err != nil {
		t.Fatal(err)
	}
	b33 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 33, ParentRoot: r32[:]}}
	if err := service.beaconDB.SaveBlock(ctx, b33); err != nil {
		t.Fatal(err)
	}
	r33, err := stateutil.BlockRoot(b33.Block)
	if err != nil {
		t.Fatal(err)
	}

	wanted := "FFG and LMD votes are not consistent"
	if err := service.verifyLMDFFGConsistent(context.Background(), 1, []byte{'a'}, r33[:]); err.Error() != wanted {
		t.Error("Did not get wanted error")
	}
}

func TestVerifyLMDFFGConsistent_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	b32 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 32}}
	if err := service.beaconDB.SaveBlock(ctx, b32); err != nil {
		t.Fatal(err)
	}
	r32, err := stateutil.BlockRoot(b32.Block)
	if err != nil {
		t.Fatal(err)
	}
	b33 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 33, ParentRoot: r32[:]}}
	if err := service.beaconDB.SaveBlock(ctx, b33); err != nil {
		t.Fatal(err)
	}
	r33, err := stateutil.BlockRoot(b33.Block)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.verifyLMDFFGConsistent(context.Background(), 1, r32[:], r33[:]); err != nil {
		t.Errorf("Could not verify LMD and FFG votes to be consistent: %v", err)
	}
}
