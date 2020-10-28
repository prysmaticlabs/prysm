package blockchain

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
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

	BlkWithOutState := testutil.NewBeaconBlock()
	BlkWithOutState.Block.Slot = 0
	require.NoError(t, db.SaveBlock(ctx, BlkWithOutState))
	BlkWithOutStateRoot, err := BlkWithOutState.Block.HashTreeRoot()
	require.NoError(t, err)

	BlkWithStateBadAtt := testutil.NewBeaconBlock()
	BlkWithStateBadAtt.Block.Slot = 1
	require.NoError(t, db.SaveBlock(ctx, BlkWithStateBadAtt))
	BlkWithStateBadAttRoot, err := BlkWithStateBadAtt.Block.HashTreeRoot()
	require.NoError(t, err)

	s := testutil.NewBeaconState()
	require.NoError(t, s.SetSlot(100*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.beaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot))

	BlkWithValidState := testutil.NewBeaconBlock()
	BlkWithValidState.Block.Slot = 2
	require.NoError(t, db.SaveBlock(ctx, BlkWithValidState))

	BlkWithValidStateRoot, err := BlkWithValidState.Block.HashTreeRoot()
	require.NoError(t, err)
	s = testutil.NewBeaconState()
	err = s.SetFork(&pb.Fork{
		Epoch:           0,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
	})
	require.NoError(t, err)
	require.NoError(t, service.beaconDB.SaveState(ctx, s, BlkWithValidStateRoot))

	tests := []struct {
		name      string
		a         *ethpb.Attestation
		wantedErr string
	}{
		{
			name:      "attestation's data slot not aligned with target vote",
			a:         &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}}},
			wantedErr: "data slot is not in the same epoch as target 1 != 0",
		},
		{
			name:      "attestation's target root not in db",
			a:         &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'A'}, 32)}}},
			wantedErr: "target root does not exist in db",
		},
		{
			name:      "no pre state for attestations's target block",
			a:         &ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}},
			wantedErr: "could not get pre state for epoch 0",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: &ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
				Root: BlkWithStateBadAttRoot[:]}}},
			wantedErr: "target epoch 100 does not match current epoch",
		},
		{
			name:      "process nil attestation",
			a:         nil,
			wantedErr: "nil attestation",
		},
		{
			name:      "process nil field (a.Data) in attestation",
			a:         &ethpb.Attestation{},
			wantedErr: "nil attestation.Data field",
		},
		{
			name: "process nil field (a.Target) in attestation",
			a: &ethpb.Attestation{
				Data: &ethpb.AttestationData{
					BeaconBlockRoot: make([]byte, 32),
					Target:          nil,
					Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
				},
				AggregationBits: make([]byte, 1),
				Signature:       make([]byte, 96),
			},
			wantedErr: "nil attestation.Data.Target field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.onAttestation(ctx, tt.a)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
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

	s := testutil.NewBeaconState()
	err = s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'A'}, 32)})
	require.NoError(t, err)
	val := &ethpb.Validator{
		PublicKey:             bytesutil.PadTo([]byte("foo"), 48),
		WithdrawalCredentials: bytesutil.PadTo([]byte("bar"), 32),
	}
	err = s.SetValidators([]*ethpb.Validator{val})
	require.NoError(t, err)
	err = s.SetBalances([]uint64{0})
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

	s1, err = service.checkpointStateCache.StateByCheckpoint(cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	s2, err = service.checkpointStateCache.StateByCheckpoint(cp2)
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
	checkpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("hi"), 32)}
	require.NoError(t, service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root)))
	returned, err := service.getAttPreState(ctx, checkpoint)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), checkpoint.Epoch*params.BeaconConfig().SlotsPerEpoch, "Incorrectly returned base state")

	cached, err := service.checkpointStateCache.StateByCheckpoint(checkpoint)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), cached.Slot(), "State should have been cached")

	epoch = uint64(2)
	newCheckpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("bye"), 32)}
	require.NoError(t, service.beaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root)))
	returned, err = service.getAttPreState(ctx, newCheckpoint)
	require.NoError(t, err)
	s, err := helpers.StartSlot(newCheckpoint.Epoch)
	require.NoError(t, err)
	baseState, err = state.ProcessSlots(ctx, baseState, s)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), baseState.Slot(), "Incorrectly returned base state")

	cached, err = service.checkpointStateCache.StateByCheckpoint(newCheckpoint)
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

	nowTime := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, 32)}))
}

func TestAttEpoch_MatchCurrentEpoch(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	nowTime := params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Epoch: 1}))
}

func TestAttEpoch_NotMatch(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	nowTime := 2 * params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot
	err = service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, 32)})
	assert.ErrorContains(t, "target epoch 0 does not match current epoch 2 or prev epoch 1", err)
}

func TestVerifyBeaconBlock_NoBlock(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	d := &ethpb.AttestationData{
		BeaconBlockRoot: make([]byte, 32),
		Target:          &ethpb.Checkpoint{Root: make([]byte, 32)},
		Source:          &ethpb.Checkpoint{Root: make([]byte, 32)},
	}
	assert.ErrorContains(t, "beacon block 0x000000000000 does not exist", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_futureBlock(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	r, err := b.Block.HashTreeRoot()
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

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b))
	r, err := b.Block.HashTreeRoot()
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

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
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

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)
	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.verifyLMDFFGConsistent(context.Background(), 1, r32[:], r33[:])
	assert.NoError(t, err, "Could not verify LMD and FFG votes to be consistent")
}

func TestVerifyFinalizedConsistency_InconsistentRoot(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1}

	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.ErrorContains(t, "Root and finalized store are not consistent", err)
}

func TestVerifyFinalizedConsistency_OK(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1, Root: r32[:]}

	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.beaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}

func TestVerifyFinalizedConsistency_IsCanonical(t *testing.T) {
	ctx := context.Background()
	db, _ := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: db, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1, Root: r32[:]}

	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, service.forkChoiceStore.ProcessBlock(ctx, b32.Block.Slot, r32, [32]byte{}, [32]byte{}, 0, 0))
	require.NoError(t, service.forkChoiceStore.ProcessBlock(ctx, b33.Block.Slot, r33, r32, [32]byte{}, 0, 0))

	_, err = service.forkChoiceStore.Head(ctx, 0, r32, []uint64{}, 0)
	require.NoError(t, err)
	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}
