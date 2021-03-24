package blockchain

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
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
	"github.com/prysmaticlabs/prysm/shared/timeutils"
)

func TestStore_OnAttestation_ErrorConditions(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB:        beaconDB,
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		StateGen:        stategen.New(beaconDB),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	_, err = blockTree1(t, beaconDB, []byte{'g'})
	require.NoError(t, err)

	BlkWithOutState := testutil.NewBeaconBlock()
	BlkWithOutState.Block.Slot = 0
	require.NoError(t, beaconDB.SaveBlock(ctx, BlkWithOutState))
	BlkWithOutStateRoot, err := BlkWithOutState.Block.HashTreeRoot()
	require.NoError(t, err)

	BlkWithStateBadAtt := testutil.NewBeaconBlock()
	BlkWithStateBadAtt.Block.Slot = 1
	require.NoError(t, beaconDB.SaveBlock(ctx, BlkWithStateBadAtt))
	BlkWithStateBadAttRoot, err := BlkWithStateBadAtt.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(100*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot))

	BlkWithValidState := testutil.NewBeaconBlock()
	BlkWithValidState.Block.Slot = 2
	require.NoError(t, beaconDB.SaveBlock(ctx, BlkWithValidState))

	BlkWithValidStateRoot, err := BlkWithValidState.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err = testutil.NewBeaconState()
	require.NoError(t, err)
	err = s.SetFork(&pb.Fork{
		Epoch:           0,
		CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
		PreviousVersion: params.BeaconConfig().GenesisForkVersion,
	})
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, BlkWithValidStateRoot))

	tests := []struct {
		name      string
		a         *ethpb.Attestation
		wantedErr string
	}{
		{
			name:      "attestation's data slot not aligned with target vote",
			a:         testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}}}),
			wantedErr: "slot 32 does not match target epoch 0",
		},
		{
			name:      "no pre state for attestations's target block",
			a:         testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}}),
			wantedErr: "could not get pre state for epoch 0",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: testutil.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
				Root: BlkWithStateBadAttRoot[:]}}}),
			wantedErr: "target epoch 100 does not match current epoch",
		},
		{
			name:      "process nil attestation",
			a:         nil,
			wantedErr: "attestation can't be nil",
		},
		{
			name:      "process nil field (a.Data) in attestation",
			a:         &ethpb.Attestation{},
			wantedErr: "attestation's data can't be nil",
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
			wantedErr: "attestation's target can't be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.onAttestation(ctx, tt.a)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStore_OnAttestation_Ok(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB:        beaconDB,
		ForkChoiceStore: protoarray.New(0, 0, [32]byte{}),
		StateGen:        stategen.New(beaconDB),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)
	genesisState, pks := testutil.DeterministicGenesisState(t, 64)
	require.NoError(t, genesisState.SetGenesisTime(uint64(timeutils.Now().Unix())-params.BeaconConfig().SecondsPerSlot))
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	att, err := testutil.GenerateAttestations(genesisState, pks, 1, 0, false)
	require.NoError(t, err)
	tRoot := bytesutil.ToBytes32(att[0].Data.Target.Root)
	copied := genesisState.Copy()
	copied, err = state.ProcessSlots(ctx, copied, 1)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, copied, tRoot))
	require.NoError(t, service.cfg.ForkChoiceStore.ProcessBlock(ctx, 0, tRoot, tRoot, tRoot, 1, 1))
	require.NoError(t, service.onAttestation(ctx, att[0]))
}

func TestStore_SaveCheckpointState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: beaconDB,
		StateGen: stategen.New(beaconDB),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	s, err := testutil.NewBeaconState()
	require.NoError(t, err)
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
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, r))

	service.justifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.bestJustifiedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.finalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}
	service.prevFinalizedCheckpt = &ethpb.Checkpoint{Root: r[:]}

	r = bytesutil.ToBytes32([]byte{'A'})
	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, 32)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'A'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'A'}, 32)}))

	s1, err := service.getAttPreState(ctx, cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, 32)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'B'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'B'}, 32)}))
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
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'C'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &pb.StateSummary{Root: bytesutil.PadTo([]byte{'C'}, 32)}))
	s3, err := service.getAttPreState(ctx, cp3)
	require.NoError(t, err)
	assert.Equal(t, s.Slot(), s3.Slot(), "Unexpected state slot")
}

func TestStore_UpdateCheckpointState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{
		BeaconDB: beaconDB,
		StateGen: stategen.New(beaconDB),
	}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	epoch := types.Epoch(1)
	baseState, _ := testutil.DeterministicGenesisState(t, 1)
	checkpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("hi"), 32)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root)))
	returned, err := service.getAttPreState(ctx, checkpoint)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch.Mul(uint64(checkpoint.Epoch)), returned.Slot(), "Incorrectly returned base state")

	cached, err := service.checkpointStateCache.StateByCheckpoint(checkpoint)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), cached.Slot(), "State should have been cached")

	epoch = 2
	newCheckpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("bye"), 32)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root)))
	returned, err = service.getAttPreState(ctx, newCheckpoint)
	require.NoError(t, err)
	s, err := helpers.StartSlot(newCheckpoint.Epoch)
	require.NoError(t, err)
	baseState, err = state.ProcessSlots(ctx, baseState, s)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), baseState.Slot(), "Incorrectly returned base state")

	cached, err = service.checkpointStateCache.StateByCheckpoint(newCheckpoint)
	require.NoError(t, err)
	require.DeepSSZEqual(t, returned.InnerStateUnsafe(), cached.InnerStateUnsafe())
}

func TestAttEpoch_MatchPrevEpoch(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	nowTime := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, 32)}))
}

func TestAttEpoch_MatchCurrentEpoch(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	nowTime := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Epoch: 1}))
}

func TestAttEpoch_NotMatch(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	nowTime := 2 * uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	err = service.verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, 32)})
	assert.ErrorContains(t, "target epoch 0 does not match current epoch 2 or prev epoch 1", err)
}

func TestVerifyBeaconBlock_NoBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	d := testutil.HydrateAttestationData(&ethpb.AttestationData{})
	assert.ErrorContains(t, "signed beacon block can't be nil", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_futureBlock(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: r[:]}

	assert.ErrorContains(t, "could not process attestation for future block", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_OK(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = 2
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: r[:]}

	assert.NoError(t, service.verifyBeaconBlock(ctx, d), "Did not receive the wanted error")
}

func TestVerifyFinalizedConsistency_InconsistentRoot(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1}

	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.ErrorContains(t, "Root and finalized store are not consistent", err)
}

func TestVerifyFinalizedConsistency_OK(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
	service, err := NewService(ctx, cfg)
	require.NoError(t, err)

	b32 := testutil.NewBeaconBlock()
	b32.Block.Slot = 32
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b32))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.finalizedCheckpt = &ethpb.Checkpoint{Epoch: 1, Root: r32[:]}

	b33 := testutil.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, b33))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}

func TestVerifyFinalizedConsistency_IsCanonical(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	cfg := &Config{BeaconDB: beaconDB, ForkChoiceStore: protoarray.New(0, 0, [32]byte{})}
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

	require.NoError(t, service.cfg.ForkChoiceStore.ProcessBlock(ctx, b32.Block.Slot, r32, [32]byte{}, [32]byte{}, 0, 0))
	require.NoError(t, service.cfg.ForkChoiceStore.ProcessBlock(ctx, b33.Block.Slot, r33, r32, [32]byte{}, 0, 0))

	_, err = service.cfg.ForkChoiceStore.Head(ctx, 0, r32, []uint64{}, 0)
	require.NoError(t, err)
	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}
