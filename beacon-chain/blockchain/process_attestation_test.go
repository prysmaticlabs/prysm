package blockchain

import (
	"context"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	testDB "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"github.com/prysmaticlabs/prysm/time/slots"
)

func TestStore_OnAttestation_ErrorConditions_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(protoarray.New(0, 0, [32]byte{})),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	_, err = blockTree1(t, beaconDB, []byte{'g'})
	require.NoError(t, err)

	BlkWithOutState := util.NewBeaconBlock()
	BlkWithOutState.Block.Slot = 0
	wsb, err := wrapper.WrappedSignedBeaconBlock(BlkWithOutState)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	BlkWithOutStateRoot, err := BlkWithOutState.Block.HashTreeRoot()
	require.NoError(t, err)

	BlkWithStateBadAtt := util.NewBeaconBlock()
	BlkWithStateBadAtt.Block.Slot = 1
	wsb, err = wrapper.WrappedSignedBeaconBlock(BlkWithStateBadAtt)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	BlkWithStateBadAttRoot, err := BlkWithStateBadAtt.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(100*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot))

	BlkWithValidState := util.NewBeaconBlock()
	BlkWithValidState.Block.Slot = 2
	wsb, err = wrapper.WrappedSignedBeaconBlock(BlkWithValidState)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))

	BlkWithValidStateRoot, err := BlkWithValidState.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err = util.NewBeaconState()
	require.NoError(t, err)
	err = s.SetFork(&ethpb.Fork{
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
			a:         util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}}}),
			wantedErr: "slot 32 does not match target epoch 0",
		},
		{
			name:      "no pre state for attestations's target block",
			a:         util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}}),
			wantedErr: "could not get pre state for epoch 0",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
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
					BeaconBlockRoot: make([]byte, fieldparams.RootLength),
					Target:          nil,
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
				AggregationBits: make([]byte, 1),
				Signature:       make([]byte, 96),
			},
			wantedErr: "attestation's target can't be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.OnAttestation(ctx, tt.a)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStore_OnAttestation_ErrorConditions_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithForkChoiceStore(doublylinkedtree.New(0, 0)),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	_, err = blockTree1(t, beaconDB, []byte{'g'})
	require.NoError(t, err)

	BlkWithOutState := util.NewBeaconBlock()
	BlkWithOutState.Block.Slot = 0
	wsb, err := wrapper.WrappedSignedBeaconBlock(BlkWithOutState)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	BlkWithOutStateRoot, err := BlkWithOutState.Block.HashTreeRoot()
	require.NoError(t, err)

	BlkWithStateBadAtt := util.NewBeaconBlock()
	BlkWithStateBadAtt.Block.Slot = 1
	wsb, err = wrapper.WrappedSignedBeaconBlock(BlkWithStateBadAtt)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))
	BlkWithStateBadAttRoot, err := BlkWithStateBadAtt.Block.HashTreeRoot()
	require.NoError(t, err)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	require.NoError(t, s.SetSlot(100*params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, BlkWithStateBadAttRoot))

	BlkWithValidState := util.NewBeaconBlock()
	BlkWithValidState.Block.Slot = 2
	wsb, err = wrapper.WrappedSignedBeaconBlock(BlkWithValidState)
	require.NoError(t, err)
	require.NoError(t, beaconDB.SaveBlock(ctx, wsb))

	BlkWithValidStateRoot, err := BlkWithValidState.Block.HashTreeRoot()
	require.NoError(t, err)
	s, err = util.NewBeaconState()
	require.NoError(t, err)
	err = s.SetFork(&ethpb.Fork{
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
			a:         util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Root: make([]byte, 32)}}}),
			wantedErr: "slot 32 does not match target epoch 0",
		},
		{
			name:      "no pre state for attestations's target block",
			a:         util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{Root: BlkWithOutStateRoot[:]}}}),
			wantedErr: "could not get pre state for epoch 0",
		},
		{
			name: "process attestation doesn't match current epoch",
			a: util.HydrateAttestation(&ethpb.Attestation{Data: &ethpb.AttestationData{Slot: 100 * params.BeaconConfig().SlotsPerEpoch, Target: &ethpb.Checkpoint{Epoch: 100,
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
					BeaconBlockRoot: make([]byte, fieldparams.RootLength),
					Target:          nil,
					Source:          &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)},
				},
				AggregationBits: make([]byte, 1),
				Signature:       make([]byte, 96),
			},
			wantedErr: "attestation's target can't be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.OnAttestation(ctx, tt.a)
			if tt.wantedErr != "" {
				assert.ErrorContains(t, tt.wantedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStore_OnAttestation_Ok_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	service.SetGenesisTime(time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0))
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	att, err := util.GenerateAttestations(genesisState, pks, 1, 0, false)
	require.NoError(t, err)
	tRoot := bytesutil.ToBytes32(att[0].Data.Target.Root)
	copied := genesisState.Copy()
	copied, err = transition.ProcessSlots(ctx, copied, 1)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, copied, tRoot))
	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 0, tRoot, tRoot, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, service.OnAttestation(ctx, att[0]))
}

func TestStore_OnAttestation_Ok_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := doublylinkedtree.New(0, 0)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)
	genesisState, pks := util.DeterministicGenesisState(t, 64)
	service.SetGenesisTime(time.Unix(time.Now().Unix()-int64(params.BeaconConfig().SecondsPerSlot), 0))
	require.NoError(t, service.saveGenesisData(ctx, genesisState))
	att, err := util.GenerateAttestations(genesisState, pks, 1, 0, false)
	require.NoError(t, err)
	tRoot := bytesutil.ToBytes32(att[0].Data.Target.Root)
	copied := genesisState.Copy()
	copied, err = transition.ProcessSlots(ctx, copied, 1)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, copied, tRoot))
	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, 0, tRoot, tRoot, params.BeaconConfig().ZeroHash, 1, 1))
	require.NoError(t, service.OnAttestation(ctx, att[0]))
}

func TestStore_SaveCheckpointState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	s, err := util.NewBeaconState()
	require.NoError(t, err)
	err = s.SetFinalizedCheckpoint(&ethpb.Checkpoint{Root: bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)})
	require.NoError(t, err)
	val := &ethpb.Validator{
		PublicKey:             bytesutil.PadTo([]byte("foo"), 48),
		WithdrawalCredentials: bytesutil.PadTo([]byte("bar"), fieldparams.RootLength),
	}
	err = s.SetValidators([]*ethpb.Validator{val})
	require.NoError(t, err)
	err = s.SetBalances([]uint64{0})
	require.NoError(t, err)
	r := [32]byte{'g'}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, r))

	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetPrevFinalizedCheckpt(&ethpb.Checkpoint{Root: r[:]})

	r = bytesutil.ToBytes32([]byte{'A'})
	cp1 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'A'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: bytesutil.PadTo([]byte{'A'}, fieldparams.RootLength)}))

	s1, err := service.getAttPreState(ctx, cp1)
	require.NoError(t, err)
	assert.Equal(t, 1*params.BeaconConfig().SlotsPerEpoch, s1.Slot(), "Unexpected state slot")

	cp2 := &ethpb.Checkpoint{Epoch: 2, Root: bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'B'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: bytesutil.PadTo([]byte{'B'}, fieldparams.RootLength)}))
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
	service.store.SetJustifiedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetBestJustifiedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	service.store.SetPrevFinalizedCheckpt(&ethpb.Checkpoint{Root: r[:]})
	cp3 := &ethpb.Checkpoint{Epoch: 1, Root: bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, s, bytesutil.ToBytes32([]byte{'C'})))
	require.NoError(t, service.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{Root: bytesutil.PadTo([]byte{'C'}, fieldparams.RootLength)}))
	s3, err := service.getAttPreState(ctx, cp3)
	require.NoError(t, err)
	assert.Equal(t, s.Slot(), s3.Slot(), "Unexpected state slot")
}

func TestStore_UpdateCheckpointState(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	epoch := types.Epoch(1)
	baseState, _ := util.DeterministicGenesisState(t, 1)
	checkpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("hi"), fieldparams.RootLength)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(checkpoint.Root)))
	returned, err := service.getAttPreState(ctx, checkpoint)
	require.NoError(t, err)
	assert.Equal(t, params.BeaconConfig().SlotsPerEpoch.Mul(uint64(checkpoint.Epoch)), returned.Slot(), "Incorrectly returned base state")

	cached, err := service.checkpointStateCache.StateByCheckpoint(checkpoint)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), cached.Slot(), "State should have been cached")

	epoch = 2
	newCheckpoint := &ethpb.Checkpoint{Epoch: epoch, Root: bytesutil.PadTo([]byte("bye"), fieldparams.RootLength)}
	require.NoError(t, service.cfg.BeaconDB.SaveState(ctx, baseState, bytesutil.ToBytes32(newCheckpoint.Root)))
	returned, err = service.getAttPreState(ctx, newCheckpoint)
	require.NoError(t, err)
	s, err := slots.EpochStart(newCheckpoint.Epoch)
	require.NoError(t, err)
	baseState, err = transition.ProcessSlots(ctx, baseState, s)
	require.NoError(t, err)
	assert.Equal(t, returned.Slot(), baseState.Slot(), "Incorrectly returned base state")

	cached, err = service.checkpointStateCache.StateByCheckpoint(newCheckpoint)
	require.NoError(t, err)
	require.DeepSSZEqual(t, returned.InnerStateUnsafe(), cached.InnerStateUnsafe())
}

func TestAttEpoch_MatchPrevEpoch(t *testing.T) {
	ctx := context.Background()

	nowTime := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)}))
}

func TestAttEpoch_MatchCurrentEpoch(t *testing.T) {
	ctx := context.Background()

	nowTime := uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	require.NoError(t, verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Epoch: 1}))
}

func TestAttEpoch_NotMatch(t *testing.T) {
	ctx := context.Background()

	nowTime := 2 * uint64(params.BeaconConfig().SlotsPerEpoch) * params.BeaconConfig().SecondsPerSlot
	err := verifyAttTargetEpoch(ctx, 0, nowTime, &ethpb.Checkpoint{Root: make([]byte, fieldparams.RootLength)})
	assert.ErrorContains(t, "target epoch 0 does not match current epoch 2 or prev epoch 1", err)
}

func TestVerifyBeaconBlock_NoBlock(t *testing.T) {
	ctx := context.Background()
	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	d := util.HydrateAttestationData(&ethpb.AttestationData{})
	assert.ErrorContains(t, "signed beacon block can't be nil", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_futureBlock(t *testing.T) {
	ctx := context.Background()

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 1, BeaconBlockRoot: r[:]}

	assert.ErrorContains(t, "could not process attestation for future block", service.verifyBeaconBlock(ctx, d))
}

func TestVerifyBeaconBlock_OK(t *testing.T) {
	ctx := context.Background()

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b := util.NewBeaconBlock()
	b.Block.Slot = 2
	wsb, err := wrapper.WrappedSignedBeaconBlock(b)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r, err := b.Block.HashTreeRoot()
	require.NoError(t, err)
	d := &ethpb.AttestationData{Slot: 2, BeaconBlockRoot: r[:]}

	assert.NoError(t, service.verifyBeaconBlock(ctx, d), "Did not receive the wanted error")
}

func TestVerifyFinalizedConsistency_InconsistentRoot_ProtoArray(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := protoarray.New(0, 0, [32]byte{'a'})
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	wsb, err := wrapper.WrappedSignedBeaconBlock(b32)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 1})
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	wsb, err = wrapper.WrappedSignedBeaconBlock(b33)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.ErrorContains(t, "Root and finalized store are not consistent", err)
}

func TestVerifyFinalizedConsistency_InconsistentRoot_DoublyLinkedTree(t *testing.T) {
	ctx := context.Background()
	beaconDB := testDB.SetupDB(t)

	fcs := doublylinkedtree.New(0, 0)
	opts := []Option{
		WithDatabase(beaconDB),
		WithStateGen(stategen.New(beaconDB)),
		WithForkChoiceStore(fcs),
	}
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	wsb, err := wrapper.WrappedSignedBeaconBlock(b32)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Epoch: 1})
	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	wsb, err = wrapper.WrappedSignedBeaconBlock(b33)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.ErrorContains(t, "Root and finalized store are not consistent", err)
}

func TestVerifyFinalizedConsistency_OK(t *testing.T) {
	ctx := context.Background()

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	wsb, err := wrapper.WrappedSignedBeaconBlock(b32)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: r32[:], Epoch: 1})

	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	wsb, err = wrapper.WrappedSignedBeaconBlock(b33)
	require.NoError(t, err)
	require.NoError(t, service.cfg.BeaconDB.SaveBlock(ctx, wsb))
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}

func TestVerifyFinalizedConsistency_IsCanonical(t *testing.T) {
	ctx := context.Background()

	opts := testServiceOptsWithDB(t)
	service, err := NewService(ctx, opts...)
	require.NoError(t, err)

	b32 := util.NewBeaconBlock()
	b32.Block.Slot = 32
	r32, err := b32.Block.HashTreeRoot()
	require.NoError(t, err)

	service.store.SetFinalizedCheckpt(&ethpb.Checkpoint{Root: r32[:], Epoch: 1})

	b33 := util.NewBeaconBlock()
	b33.Block.Slot = 33
	b33.Block.ParentRoot = r32[:]
	r33, err := b33.Block.HashTreeRoot()
	require.NoError(t, err)

	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, b32.Block.Slot, r32, [32]byte{}, params.BeaconConfig().ZeroHash, 0, 0))
	require.NoError(t, service.cfg.ForkChoiceStore.InsertOptimisticBlock(ctx, b33.Block.Slot, r33, r32, params.BeaconConfig().ZeroHash, 0, 0))

	_, err = service.cfg.ForkChoiceStore.Head(ctx, 0, r32, []uint64{}, 0)
	require.NoError(t, err)
	err = service.VerifyFinalizedConsistency(context.Background(), r33[:])
	require.NoError(t, err)
}
