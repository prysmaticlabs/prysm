package blocks_test

import (
	"context"
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/aggregation"
	attaggregation "github.com/prysmaticlabs/prysm/shared/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestProcessAttestations_InclusionDelayFailure(t *testing.T) {
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				Slot:   5,
			},
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	want := fmt.Sprintf(
		"attestation slot %d + inclusion delay %d > state slot %d",
		attestations[0].Data.Slot,
		params.BeaconConfig().MinAttestationInclusionDelay,
		beaconState.Slot(),
	)
	_, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)
}

func TestProcessAttestations_NeitherCurrentNorPrevEpoch(t *testing.T) {
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}}}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch*4 + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetPreviousJustifiedCheckpoint(pfc))
	require.NoError(t, beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{}))

	want := fmt.Sprintf(
		"expected target epoch (%d) to be the previous epoch (%d) or the current epoch (%d)",
		att.Data.Target.Epoch,
		helpers.PrevEpoch(beaconState),
		helpers.CurrentEpoch(beaconState),
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)
}

func TestProcessAttestations_CurrentEpochFFGDataMismatches(t *testing.T) {
	aggBits := bitfield.NewBitlist(3)
	attestations := []*ethpb.Attestation{
		{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
				Source: &ethpb.Checkpoint{Epoch: 1, Root: make([]byte, 32)},
			},
			AggregationBits: aggBits,
		},
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+params.BeaconConfig().MinAttestationInclusionDelay))
	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.CurrentEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	_, err := blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)

	block.Body.Attestations[0].Data.Source.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint().Root,
		attestations[0].Data.Source.Root,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)
}

func TestProcessAttestations_PrevEpochFFGDataMismatches(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

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
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	pfc := beaconState.PreviousJustifiedCheckpoint()
	pfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetPreviousJustifiedCheckpoint(pfc))
	require.NoError(t, beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{}))

	want := fmt.Sprintf(
		"expected source epoch %d, received %d",
		helpers.PrevEpoch(beaconState),
		attestations[0].Data.Source.Epoch,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)

	block.Body.Attestations[0].Data.Source.Epoch = helpers.PrevEpoch(beaconState)
	block.Body.Attestations[0].Data.Target.Epoch = helpers.CurrentEpoch(beaconState)
	block.Body.Attestations[0].Data.Source.Root = []byte{}

	want = fmt.Sprintf(
		"expected source root %#x, received %#x",
		beaconState.CurrentJustifiedCheckpoint().Root,
		attestations[0].Data.Source.Root,
	)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, want, err)
}

func TestProcessAttestations_InvalidAggregationBitsLength(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(4)
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Target: &ethpb.Checkpoint{Epoch: 0}},
		AggregationBits: aggBits,
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	expected := "failed to verify aggregation bitfield: wanted participants bitfield length 3, got: 4"
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.ErrorContains(t, expected, err)
}

func TestProcessAttestations_OK(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(0, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
		},
		AggregationBits: aggBits,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att.Data.Slot, att.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices := attestationutil.AttestingIndices(att.AggregationBits, committee)
	require.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices))
	for i, indice := range attestingIndices {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, att.Data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{att},
		},
	}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.NoError(t, err)
}

func TestProcessAggregatedAttestation_OverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
	}
	aggBits1 := bitfield.NewBitlist(4)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	aggBits1.SetBitAt(2, true)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = []byte("hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices1 := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	require.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, att1.Data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggBits2 := bitfield.NewBitlist(4)
	aggBits2.SetBitAt(1, true)
	aggBits2.SetBitAt(2, true)
	aggBits2.SetBitAt(3, true)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
	}

	committee, err = helpers.BeaconCommitteeFromState(beaconState, att2.Data.Slot, att2.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices2 := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	require.NoError(t, err)
	sigs = make([]bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, att2.Data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	_, err = attaggregation.AggregatePair(att1, att2)
	assert.ErrorContains(t, aggregation.ErrBitsOverlap.Error(), err)
}

func TestProcessAggregatedAttestation_NoOverlappingBits(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 300)

	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	data := &ethpb.AttestationData{
		Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
		Target: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
	}
	aggBits1 := bitfield.NewBitlist(9)
	aggBits1.SetBitAt(0, true)
	aggBits1.SetBitAt(1, true)
	att1 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits1,
	}

	cfc := beaconState.CurrentJustifiedCheckpoint()
	cfc.Root = mockRoot[:]
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(cfc))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	committee, err := helpers.BeaconCommitteeFromState(beaconState, att1.Data.Slot, att1.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices1 := attestationutil.AttestingIndices(att1.AggregationBits, committee)
	require.NoError(t, err)
	sigs := make([]bls.Signature, len(attestingIndices1))
	for i, indice := range attestingIndices1 {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggBits2 := bitfield.NewBitlist(9)
	aggBits2.SetBitAt(2, true)
	aggBits2.SetBitAt(3, true)
	att2 := &ethpb.Attestation{
		Data:            data,
		AggregationBits: aggBits2,
	}

	committee, err = helpers.BeaconCommitteeFromState(beaconState, att2.Data.Slot, att2.Data.CommitteeIndex)
	require.NoError(t, err)
	attestingIndices2 := attestationutil.AttestingIndices(att2.AggregationBits, committee)
	require.NoError(t, err)
	sigs = make([]bls.Signature, len(attestingIndices2))
	for i, indice := range attestingIndices2 {
		sb, err := helpers.ComputeDomainAndSign(beaconState, 0, att2.Data, params.BeaconConfig().DomainBeaconAttester, privKeys[indice])
		require.NoError(t, err)
		sig, err := bls.SignatureFromBytes(sb)
		require.NoError(t, err)
		sigs[i] = sig
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()[:]

	aggregatedAtt, err := attaggregation.AggregatePair(att1, att2)
	require.NoError(t, err)
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			Attestations: []*ethpb.Attestation{aggregatedAtt},
		},
	}

	err = beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)

	_, err = blocks.ProcessAttestations(context.Background(), beaconState, block.Body)
	assert.NoError(t, err)
}

func TestProcessAttestationsNoVerify_IncorrectSlotTargetEpoch(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 1)

	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Slot:   params.BeaconConfig().SlotsPerEpoch,
			Target: &ethpb.Checkpoint{},
		},
	}
	wanted := fmt.Sprintf("data slot is not in the same epoch as target %d != %d", helpers.SlotToEpoch(att.Data.Slot), att.Data.Target.Epoch)
	_, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att)
	assert.ErrorContains(t, wanted, err)
}

func TestProcessAttestationsNoVerify_OK(t *testing.T) {
	// Attestation with an empty signature

	beaconState, _ := testutil.DeterministicGenesisState(t, 100)

	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(1, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AggregationBits: aggBits,
	}

	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]

	err := beaconState.SetSlot(beaconState.Slot() + params.BeaconConfig().MinAttestationInclusionDelay)
	require.NoError(t, err)
	ckp := beaconState.CurrentJustifiedCheckpoint()
	copy(ckp.Root, "hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(ckp))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))

	_, err = blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att)
	assert.NoError(t, err)
}

func TestProcessAttestationsNoVerify_BadAttIdx(t *testing.T) {
	beaconState, _ := testutil.DeterministicGenesisState(t, 100)
	aggBits := bitfield.NewBitlist(3)
	aggBits.SetBitAt(1, true)
	var mockRoot [32]byte
	copy(mockRoot[:], "hello-world")
	att := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			CommitteeIndex: 100,
			Source:         &ethpb.Checkpoint{Epoch: 0, Root: mockRoot[:]},
			Target:         &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		AggregationBits: aggBits,
	}
	zeroSig := [96]byte{}
	att.Signature = zeroSig[:]
	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+params.BeaconConfig().MinAttestationInclusionDelay))
	ckp := beaconState.CurrentJustifiedCheckpoint()
	copy(ckp.Root, "hello-world")
	require.NoError(t, beaconState.SetCurrentJustifiedCheckpoint(ckp))
	require.NoError(t, beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{}))
	_, err := blocks.ProcessAttestationNoVerify(context.TODO(), beaconState, att)
	require.ErrorContains(t, "committee index 100 >= committee count 1", err)
}

func TestConvertToIndexed_OK(t *testing.T) {
	helpers.ClearCache()
	validators := make([]*ethpb.Validator, 2*params.BeaconConfig().SlotsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:        5,
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)
	tests := []struct {
		aggregationBitfield    bitfield.Bitlist
		wantedAttestingIndices []uint64
	}{
		{
			aggregationBitfield:    bitfield.Bitlist{0x07},
			wantedAttestingIndices: []uint64{43, 47},
		},
		{
			aggregationBitfield:    bitfield.Bitlist{0x03},
			wantedAttestingIndices: []uint64{47},
		},
		{
			aggregationBitfield:    bitfield.Bitlist{0x01},
			wantedAttestingIndices: []uint64{},
		},
	}

	var sig [96]byte
	copy(sig[:], "signed")
	attestation := &ethpb.Attestation{
		Signature: sig[:],
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
			Target: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
	}
	for _, tt := range tests {
		attestation.AggregationBits = tt.aggregationBitfield
		wanted := &ethpb.IndexedAttestation{
			AttestingIndices: tt.wantedAttestingIndices,
			Data:             attestation.Data,
			Signature:        attestation.Signature,
		}

		committee, err := helpers.BeaconCommitteeFromState(state, attestation.Data.Slot, attestation.Data.CommitteeIndex)
		require.NoError(t, err)
		ia := attestationutil.ConvertToIndexed(context.Background(), attestation, committee)
		assert.DeepEqual(t, wanted, ia, "Convert attestation to indexed attestation didn't result as wanted")
	}
}

func TestVerifyIndexedAttestation_OK(t *testing.T) {
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	require.NoError(t, err)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	require.NoError(t, err)
	tests := []struct {
		attestation *ethpb.IndexedAttestation
	}{
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 2,
				},
			},
			AttestingIndices: []uint64{1},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 1,
				},
			},
			AttestingIndices: []uint64{47, 99, 101},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 4,
				},
			},
			AttestingIndices: []uint64{21, 72},
		}},
		{attestation: &ethpb.IndexedAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{
					Epoch: 7,
				},
			},
			AttestingIndices: []uint64{100, 121, 122},
		}},
	}

	for _, tt := range tests {
		var sig []bls.Signature
		for _, idx := range tt.attestation.AttestingIndices {
			sb, err := helpers.ComputeDomainAndSign(state, tt.attestation.Data.Target.Epoch, tt.attestation.Data, params.BeaconConfig().DomainBeaconAttester, keys[idx])
			require.NoError(t, err)
			validatorSig, err := bls.SignatureFromBytes(sb)
			require.NoError(t, err)
			sig = append(sig, validatorSig)
		}
		aggSig := bls.AggregateSignatures(sig)
		marshalledSig := aggSig.Marshal()

		tt.attestation.Signature = marshalledSig

		err = blocks.VerifyIndexedAttestation(context.Background(), state, tt.attestation)
		assert.NoError(t, err, "Failed to verify indexed attestation")
	}
}

func TestValidateIndexedAttestation_AboveMaxLength(t *testing.T) {
	indexedAtt1 := &ethpb.IndexedAttestation{
		AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+5),
	}

	for i := uint64(0); i < params.BeaconConfig().MaxValidatorsPerCommittee+5; i++ {
		indexedAtt1.AttestingIndices[i] = i
		indexedAtt1.Data = &ethpb.AttestationData{
			Target: &ethpb.Checkpoint{
				Epoch: i,
			},
		}
	}

	want := "validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE"
	err := blocks.VerifyIndexedAttestation(context.Background(), &stateTrie.BeaconState{}, indexedAtt1)
	assert.ErrorContains(t, want, err)
}

func TestVerifyAttestations_VerifiesMultipleAttestations(t *testing.T) {
	ctx := context.Background()
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	require.NoError(t, err)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	comm1, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 0 /*committeeIndex*/)
	require.NoError(t, err)
	att1 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm1))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		Signature: nil,
	}
	domain, err := helpers.Domain(st.Fork(), st.Fork().Epoch, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	require.NoError(t, err)
	root, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err)
	var sigs []bls.Signature
	for i, u := range comm1 {
		att1.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()

	comm2, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 1 /*committeeIndex*/)
	require.NoError(t, err)
	att2 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm2))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 1,
		},
		Signature: nil,
	}
	root, err = helpers.ComputeSigningRoot(att2.Data, domain)
	require.NoError(t, err)
	sigs = nil
	for i, u := range comm2 {
		att2.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()

	require.NoError(t, blocks.VerifyAttestations(ctx, st, []*ethpb.Attestation{att1, att2}))
}

func TestVerifyAttestations_HandlesPlannedFork(t *testing.T) {
	// In this test, att1 is from the prior fork and att2 is from the new fork.
	ctx := context.Background()
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	require.NoError(t, err)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       35,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           1,
			CurrentVersion:  []byte{0, 1, 2, 3},
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	comm1, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 0 /*committeeIndex*/)
	require.NoError(t, err)
	att1 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm1))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		Signature: nil,
	}
	prevDomain, err := helpers.Domain(st.Fork(), st.Fork().Epoch-1, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	require.NoError(t, err)
	root, err := helpers.ComputeSigningRoot(att1.Data, prevDomain)
	require.NoError(t, err)
	var sigs []bls.Signature
	for i, u := range comm1 {
		att1.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()

	comm2, err := helpers.BeaconCommitteeFromState(st, 1*params.BeaconConfig().SlotsPerEpoch+1 /*slot*/, 1 /*committeeIndex*/)
	require.NoError(t, err)
	att2 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm2))),
		Data: &ethpb.AttestationData{
			Slot:           1*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex: 1,
		},
		Signature: nil,
	}
	currDomain, err := helpers.Domain(st.Fork(), st.Fork().Epoch, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	root, err = helpers.ComputeSigningRoot(att2.Data, currDomain)
	require.NoError(t, err)
	sigs = nil
	for i, u := range comm2 {
		att2.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()

	require.NoError(t, blocks.VerifyAttestations(ctx, st, []*ethpb.Attestation{att1, att2}))
}

func TestRetrieveAttestationSignatureSet_VerifiesMultipleAttestations(t *testing.T) {
	ctx := context.Background()
	numOfValidators := 4 * params.BeaconConfig().SlotsPerEpoch
	validators := make([]*ethpb.Validator, numOfValidators)
	_, keys, err := testutil.DeterministicDepositsAndKeys(numOfValidators)
	require.NoError(t, err)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: keys[i].PublicKey().Marshal(),
		}
	}

	st, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Slot:       5,
		Validators: validators,
		Fork: &pb.Fork{
			Epoch:           0,
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})

	comm1, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 0 /*committeeIndex*/)
	require.NoError(t, err)
	att1 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm1))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 0,
		},
		Signature: nil,
	}
	domain, err := helpers.Domain(st.Fork(), st.Fork().Epoch, params.BeaconConfig().DomainBeaconAttester, st.GenesisValidatorRoot())
	require.NoError(t, err)
	root, err := helpers.ComputeSigningRoot(att1.Data, domain)
	require.NoError(t, err)
	var sigs []bls.Signature
	for i, u := range comm1 {
		att1.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att1.Signature = bls.AggregateSignatures(sigs).Marshal()

	comm2, err := helpers.BeaconCommitteeFromState(st, 1 /*slot*/, 1 /*committeeIndex*/)
	require.NoError(t, err)
	att2 := &ethpb.Attestation{
		AggregationBits: bitfield.NewBitlist(uint64(len(comm2))),
		Data: &ethpb.AttestationData{
			Slot:           1,
			CommitteeIndex: 1,
		},
		Signature: nil,
	}
	root, err = helpers.ComputeSigningRoot(att2.Data, domain)
	require.NoError(t, err)
	sigs = nil
	for i, u := range comm2 {
		att2.AggregationBits.SetBitAt(uint64(i), true)
		sigs = append(sigs, keys[u].Sign(root[:]))
	}
	att2.Signature = bls.AggregateSignatures(sigs).Marshal()

	set, err := blocks.AttestationSignatureSet(ctx, st, []*ethpb.Attestation{att1, att2})
	require.NoError(t, err)
	verified, err := set.Verify()
	require.NoError(t, err)
	assert.Equal(t, true, verified, "Multiple signatures were unable to be verified.")
}
