package altair_test

import (
	"context"
	"fmt"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/prysmaticlabs/go-bitfield"
	altair "github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateAltair "github.com/prysmaticlabs/prysm/beacon-chain/state/v2"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*ethpb.Attestation{
		testutil.HydrateAttestation(&ethpb.Attestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				Slot:   5,
			},
		}),
	}
	b := testutil.NewBeaconBlockAltair()
	b.Block = &ethpb.BeaconBlockAltair{
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 100)

	want := fmt.Sprintf(
		"attestation slot %d + inclusion delay %d > state slot %d",
		attestations[0].Data.Slot,
		params.BeaconConfig().MinAttestationInclusionDelay,
		beaconState.Slot(),
	)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
}

func TestProcessAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	att := testutil.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}}})

	b := testutil.NewBeaconBlockAltair()
	b.Block = &ethpb.BeaconBlockAltair{
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: []*ethpb.Attestation{att},
		},
	}
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 100)
	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetPreviousJustifiedCheckpoint(pfc))

	want := fmt.Sprintf(
		"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
		att.Data.Target.Epoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
}

func TestProcessAttestations_CurrentEpochFFGDataMismatches(t *testing.T) {
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
			},
			AggregationBits: bitfield.Bitlist{0x09},
		},
	}
	b := testutil.NewBeaconBlockAltair()
	b.Block = &ethpb.BeaconBlockAltair{
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 100)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+params.BeaconConfig().MinAttestationInclusionDelay))
	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))

	want := "source check point not equal to current justified checkpoint"
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
	b.Block.Body.Attestations[0].Data.Source.Epoch = helpers.CurrentEpoch(beaconState)
	b.Block.Body.Attestations[0].Data.Source.Root = []byte{}
	wsb, err = wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
}

func TestProcessAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
				Target: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
				Slot:   params.BeaconConfig().SlotsPerEpoch,
			},
			AggregationBits: aggBits,
		},
	}
	b := testutil.NewBeaconBlockAltair()
	b.Block = &ethpb.BeaconBlockAltair{
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: attestations,
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + 2*params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, err)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetPreviousJustifiedCheckpoint(pfc))

	want := "source check point not equal to previous justified checkpoint"
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
	b.Block.Body.Attestations[0].Data.Source.Epoch = helpers.PrevEpoch(beaconState)
	b.Block.Body.Attestations[0].Data.Target.Epoch = helpers.PrevEpoch(beaconState)
	b.Block.Body.Attestations[0].Data.Source.Root = []byte{}
	wsb, err = wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, want, err)
}

func TestProcessAttestations_InvalidAggregationBitsLength(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 100)

	aggBits := bitfield.NewBitlist(4)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: aggBits,
	}

	b := testutil.NewBeaconBlockAltair()
	b.Block = &ethpb.BeaconBlockAltair{
		Body: &ethpb.BeaconBlockBodyAltair{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))

	expected := "failed to verify aggregation bitfield: wanted participants bitfield length 3, got: 4"
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.ErrorContains(t, expected, err)
}

func TestProcessAttestations_OK(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisStateAltair(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := testutil.HydrateAttestation(&ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Root: mockRoot[:]},
			Target: &ethpb.Checkpoint{Root: mockRoot[:]},
		},
		AggregationBits: aggBits,
	})

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, att.Data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()

	block := testutil.NewBeaconBlockAltair()
	block.Block.Body.Attestations = []*ethpb.Attestation{att}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	wsb, err := wrapper.WrappedAltairSignedBeaconBlock(block)
	require.NoError(t, err)
	_, err = altair.ProcessAttestations(context.Background(), beaconState, wsb)
	require.NoError(t, err)
}

func TestProcessAttestationNoVerify_SourceTargetHead(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, 64)
	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	aggBits := bitfield.NewBitlist(2)
	aggBits.SetBitAt(0, true)
	aggBits.SetBitAt(1, true)
	r, err := helpers.BlockRootAtSlot(beaconState, 0)
	require.NoError(t, err)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: r,
			Source:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			Target:          &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AggregationBits: aggBits,
	}
	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	ckp := beaconState.CurrentJustifiedCheckpoint()
	copy(ckp.Root, make([]byte, 32))
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(ckp))

	beaconState, err = altair.ProcessAttestationNoVerifySignature(context.Background(), beaconState, att)
	require.NoError(t, err)

	p, err := beaconState.CurrentEpochParticipation()
	require.NoError(t, err)

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	require.NoError(t, err)
	indices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	for _, index := range indices {
		require.Equal(t, true, altair.HasValidatorFlag(p[index], params.BeaconConfig().TimelyHeadFlagIndex))
		require.Equal(t, true, altair.HasValidatorFlag(p[index], params.BeaconConfig().TimelyTargetFlagIndex))
		require.Equal(t, true, altair.HasValidatorFlag(p[index], params.BeaconConfig().TimelySourceFlagIndex))
	}
}

func TestValidatorFlag_Has(t *testing.T) {
	tests := []struct {
		name     string
		set      uint8
		expected []uint8
	}{
		{name: "none",
			set:      0,
			expected: []uint8{},
		},
		{
			name:     "source",
			set:      1,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex},
		},
		{
			name:     "target",
			set:      2,
			expected: []uint8{params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "head",
			set:      4,
			expected: []uint8{params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:     "source, target",
			set:      3,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "source, head",
			set:      5,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:     "target, head",
			set:      6,
			expected: []uint8{params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "source, target, head",
			set:      7,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, f := range tt.expected {
				require.Equal(t, true, altair.HasValidatorFlag(tt.set, f))
			}
		})
	}
}

func TestValidatorFlag_Add(t *testing.T) {
	tests := []struct {
		name          string
		set           []uint8
		expectedTrue  []uint8
		expectedFalse []uint8
	}{
		{name: "none",
			set:           []uint8{},
			expectedTrue:  []uint8{},
			expectedFalse: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex},
			expectedFalse: []uint8{params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source, target",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
			expectedFalse: []uint8{params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source, target, head",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
			expectedFalse: []uint8{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := uint8(0)
			for _, f := range tt.set {
				b = altair.AddValidatorFlag(b, f)
			}
			for _, f := range tt.expectedFalse {
				require.Equal(t, false, altair.HasValidatorFlag(b, f))
			}
			for _, f := range tt.expectedTrue {
				require.Equal(t, true, altair.HasValidatorFlag(b, f))
			}
		})
	}
}

func TestFuzzProcessAttestationsNoVerify_10000(t *testing.T) {
	fuzzer := fuzz.NewWithSeed(0)
	state := &ethpb.BeaconStateAltair{}
	b := &ethpb.SignedBeaconBlockAltair{Block: &ethpb.BeaconBlockAltair{}}
	ctx := context.Background()
	for i := 0; i < 10000; i++ {
		fuzzer.Fuzz(state)
		fuzzer.Fuzz(b)
		if b.Block == nil {
			b.Block = &ethpb.BeaconBlockAltair{}
		}
		s, err := stateAltair.InitializeFromProtoUnsafe(state)
		require.NoError(t, err)
		wsb, err := wrapper.WrappedAltairSignedBeaconBlock(b)
		require.NoError(t, err)
		r, err := altair.ProcessAttestationsNoVerifySignature(ctx, s, wsb)
		if err != nil && r != nil {
			t.Fatalf("return value should be nil on err. found: %v on error: %v for state: %v and block: %v", r, err, state, b)
		}
	}
}

func TestAttestationParticipationFlagIndices(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, beaconState.SetSlot(1))
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex

	tests := []struct {
		name                 string
		inputState           state.BeaconState
		inputData            *ethpb.AttestationData
		inputDelay           types.Slot
		participationIndices map[uint8]bool
	}{
		{
			name: "none",
			inputState: func() state.BeaconState {
				return beaconState
			}(),
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
				Target: &ethpb.Checkpoint{},
			},
			inputDelay:           params.BeaconConfig().SlotsPerEpoch,
			participationIndices: map[uint8]bool{},
		},
		{
			name: "participated source",
			inputState: func() state.BeaconState {
				return beaconState
			}(),
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
				Target: &ethpb.Checkpoint{},
			},
			inputDelay: types.Slot(mathutil.IntegerSquareRoot(uint64(cfg.SlotsPerEpoch)) - 1),
			participationIndices: map[uint8]bool{
				sourceFlagIndex: true,
			},
		},
		{
			name: "participated source and target",
			inputState: func() state.BeaconState {
				return beaconState
			}(),
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
				Target: &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
			},
			inputDelay: types.Slot(mathutil.IntegerSquareRoot(uint64(cfg.SlotsPerEpoch)) - 1),
			participationIndices: map[uint8]bool{
				sourceFlagIndex: true,
				targetFlagIndex: true,
			},
		},
		{
			name: "participated source and target and head",
			inputState: func() state.BeaconState {
				return beaconState
			}(),
			inputData: &ethpb.AttestationData{
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
				Source:          &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
				Target:          &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
			},
			inputDelay: 1,
			participationIndices: map[uint8]bool{
				sourceFlagIndex: true,
				targetFlagIndex: true,
				headFlagIndex:   true,
			},
		},
	}

	for _, test := range tests {
		flagIndices, err := altair.AttestationParticipationFlagIndices(test.inputState, test.inputData, test.inputDelay)
		require.NoError(t, err)
		require.DeepEqual(t, test.participationIndices, flagIndices)
	}
}

func TestMatchingStatus(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, beaconState.SetSlot(1))
	tests := []struct {
		name          string
		inputState    state.BeaconState
		inputData     *ethpb.AttestationData
		inputCheckpt  *ethpb.Checkpoint
		matchedSource bool
		matchedTarget bool
		matchedHead   bool
	}{
		{
			name:       "non matched",
			inputState: beaconState,
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{},
			},
			inputCheckpt: &ethpb.Checkpoint{},
		},
		{
			name:       "source matched",
			inputState: beaconState,
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{},
			},
			inputCheckpt:  &ethpb.Checkpoint{},
			matchedSource: true,
		},
		{
			name:       "target matched",
			inputState: beaconState,
			inputData: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{Epoch: 1},
				Target: &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
			},
			inputCheckpt:  &ethpb.Checkpoint{},
			matchedTarget: true,
		},
		{
			name:       "head matched",
			inputState: beaconState,
			inputData: &ethpb.AttestationData{
				Source:          &ethpb.Checkpoint{Epoch: 1},
				Target:          &ethpb.Checkpoint{},
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
			},
			inputCheckpt: &ethpb.Checkpoint{},
			matchedHead:  true,
		},
		{
			name:       "everything matched",
			inputState: beaconState,
			inputData: &ethpb.AttestationData{
				Source:          &ethpb.Checkpoint{},
				Target:          &ethpb.Checkpoint{Root: params.BeaconConfig().ZeroHash[:]},
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
			},
			inputCheckpt:  &ethpb.Checkpoint{},
			matchedSource: true,
			matchedTarget: true,
			matchedHead:   true,
		},
	}

	for _, test := range tests {
		src, tgt, head, err := altair.MatchingStatus(test.inputState, test.inputData, test.inputCheckpt)
		require.NoError(t, err)
		require.Equal(t, test.matchedSource, bool(src))
		require.Equal(t, test.matchedTarget, bool(tgt))
		require.Equal(t, test.matchedHead, bool(head))
	}
}
