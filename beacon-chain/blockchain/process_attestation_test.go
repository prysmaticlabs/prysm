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
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
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
	require.NoError(t, err)

	_, err = blockTree1(db, []byte{'g'})
	require.NoError(t, err)

	BlkWithOutState := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 0}}
	require.NoError(t, db.SaveBlock(ctx, BlkWithOutState))
	BlkWithOutStateRoot, err := stateutil.BlockRoot(BlkWithOutState.Block)
	require.NoError(t, err)

	BlkWithStateBadAtt := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 1}}
	require.NoError(t, db.SaveBlock(ctx, BlkWithStateBadAtt))
	BlkWithStateBadAttRoot, err := stateutil.BlockRoot(BlkWithStateBadAtt.Block)
	require.NoError(t, err)

	s := testutil.NewBeaconState()
	require.NoError(t, s.SetSlot(100*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.beaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot))

	BlkWithValidState := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	require.NoError(t, db.SaveBlock(ctx, BlkWithValidState))

	BlkWithValidStateRoot, err := stateutil.BlockRoot(BlkWithValidState.Block)
	require.NoError(t, err)
	s = testutil.NewBeaconState()
	if err := s.SetFork(&pb.Fork{
		Epoch:           0,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
	}); err != nil {
		t.Fatal(err)
	}
	require.NoError(t, service.beaconDB.SaveState(ctx, s, BlkWithValidStateRoot))

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
	require.NoError(t, err)

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
	require.NoError(t, err)
	r := [32]byte{'g'}
	require.NoError(t, service.beaconDB.SaveState(ctx, s, r))

	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}

	r = bytesutil.ToBytes32([]byte{'A'})
	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, 32)}
	require.NoError(t, service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'A'})))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'A'}, 32)}))

	s1, err := service.getAttPreState(ctx, cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}
	require.NoError(t, service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'B'})))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'B'}, 32)}))
	s2, err := service.getAttPreState(ctx, cp2)
	require.NoError(t, err)
	assert.Equal(t, 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot(), "Unexpected state slot")

	s1, err = service.getAttPreState(ctx, cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	s1, err = service.checkpointState.StateByCheckpoint(cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	s2, err = service.checkpointState.StateByCheckpoint(cp2)
	require.NoError(t, err)
	assert.Equal(t, 2*params.BeaconConfig().SlotsPerEpoch, s2.Slot(), "Unexpected state slot")

	require.NoError(t, s.SetSlot(params.BeaconConfig().SlotsPerEpoch+1))
	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	cp3 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, 32)}
	require.NoError(t, service.beaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'C'})))
	require.NoError(t, service.beaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'C'}, 32)}))
	s3, err := service.getAttPreState(ctx, cp3)
	require.NoError(t, err)
	assert.Equal(t, s.Slot(), s3.Slot(), "Unexpected state slot")
}

func TestStore_UpdateCheckpointState(t *testing.T) {
	ctx := context.Background()
	db, sc := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: db,
		StateGen: stategen.New(db, sc),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	epoch := uint64(1)
	baseState, _ := testutil.DeterministicGenesisState(t, 1)
	require.NoError(t, baseState.SetSlot(epoch*params.BeaconConfig().SlotsPerEpoch))
	checkpoint := &ethpb.Checkpoint{Epoch: epoch}
	require.NoError(t, service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root)))
	returned, err := service.getAttPreState(ctx, checkpoint)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), baseState.Slot(), "Incorrectly returned base state")

	cached, err := service.checkpointState.StateByCheckpoint(checkpoint)
	require.NoError(t, err)
	assert.NotNil(t, cached, "State should have been cached")

	epoch = uint64(2)
	newCheckpoint := &ethpb.Checkpoint{Epoch: epoch}
	require.NoError(t, service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root)))
	returned, err = service.getAttPreState(ctx, newCheckpoint)
	require.NoError(t, err)
	baseState, err = state.ProcessSlots(ctx, baseState, helpers.StartSlot(newCheckpoint.Epoch))
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), baseState.Slot(), "Incorrectly returned base state")

	cached, err = service.checkpointState.StateByCheckpoint(newCheckpoint)
	require.NoError(t, err)
	if !proto.Equal(returned.InnerStateUnsafe(), cached.InnerStateUnsafe()) {
		t.Error("Incorrectly cached base state")
	}
}

func TestAttEpoch_MatchPrevEpoch(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	d := &ethpb.AttestationData{}
	assert.ErrorContains(t, "beacon block  does not exist", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_futureBlock(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	r, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: r[:]}

	assert.ErrorContains(t, "could not process attestation for future block", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 2}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	r, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: r[:]}

	assert.NoError(t, service.verifyBeaconBlock(ctx, d), "Did not receive the wanted error")
}

func TestVerifyLMDFFGConsistent_NotOK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 32}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := stateutil.BlockRoot(b32.Block)
	require.NoError(t, err)
	b33 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 33, ParentRoot: r32[:]}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := stateutil.BlockRoot(b33.Block)
	require.NoError(t, err)

	wanted := "FFG and LMD votes are not consistent"
	assert.ErrorContains(t, wanted, service.verifyLMDFFGConsistent(context.Background(), 1, []byte{'a'}, r33[:]))
}

func TestVerifyLMDFFGConsistent_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 32}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := stateutil.BlockRoot(b32.Block)
	require.NoError(t, err)
	b33 := &ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 33, ParentRoot: r32[:]}}
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := stateutil.BlockRoot(b33.Block)
	require.NoError(t, err)

	err = service.verifyLMDFFGConsistent(context.Background(), 1, r32[:], r33[:])
	assert.NoError(t, err, "Could not verify LMD and FFG votes to be consistent")
}
