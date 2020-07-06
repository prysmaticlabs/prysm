package blocks_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestSlashableAttestationData_CanSlash(t *testing.T) {
	att1 := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1},
		Source: &ethpb.Checkpoint{Root: []byte{'A'}},
	}
	att2 := &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1},
		Source: &ethpb.Checkpoint{Root: []byte{'B'}},
	}
	if !blocks.IsSlashableAttestationData(att1, att2) {
		t.Error("atts should have been slashable")
	}
	att1.Target.Epoch = 4
	att1.Source.Epoch = 2
	att2.Source.Epoch = 3
	if !blocks.IsSlashableAttestationData(att1, att2) {
		t.Error("atts should have been slashable")
	}
}

func TestProcessAttesterSlashings_DataNotSlashable(t *testing.T) {
	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 1},
				},
			},
		},
	}
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}
	want := fmt.Sprint("attestations are not slashable")

	_, err = blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_IndexedAttestationFailedToVerify(t *testing.T) {
	registry := []*ethpb.Validator{}
	currentSlot := uint64(0)

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 1},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
			Attestation_2: &ethpb.IndexedAttestation{
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{Epoch: 0},
					Target: &ethpb.Checkpoint{Epoch: 0},
				},
				AttestingIndices: make([]uint64, params.BeaconConfig().MaxValidatorsPerCommittee+1),
			},
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	want := fmt.Sprint("validator indices count exceeds MAX_VALIDATORS_PER_COMMITTEE")
	_, err = blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessAttesterSlashings_AppliesCorrectStatus(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	for _, vv := range beaconState.Validators() {
		vv.WithdrawableEpoch = 1 * params.BeaconConfig().SlotsPerEpoch
	}

	att1 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 1},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AttestingIndices: []uint64{0, 1},
	}
	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconAttester, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(att1.Data, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	sig0 := privKeys[0].Sign(signingRoot[:])
	sig1 := privKeys[1].Sign(signingRoot[:])
	aggregateSig := bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att1.Signature = aggregateSig.Marshal()[:]

	att2 := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{Epoch: 0},
			Target: &ethpb.Checkpoint{Epoch: 0},
		},
		AttestingIndices: []uint64{0, 1},
	}
	signingRoot, err = helpers.ComputeSigningRoot(att2.Data, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	sig0 = privKeys[0].Sign(signingRoot[:])
	sig1 = privKeys[1].Sign(signingRoot[:])
	aggregateSig = bls.AggregateSignatures([]bls.Signature{sig0, sig1})
	att2.Signature = aggregateSig.Marshal()[:]

	slashings := []*ethpb.AttesterSlashing{
		{
			Attestation_1: att1,
			Attestation_2: att2,
		},
	}

	currentSlot := 2 * params.BeaconConfig().SlotsPerEpoch
	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			AttesterSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessAttesterSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatal(err)
	}
	newRegistry := newState.Validators()

	// Given the intersection of slashable indices is [1], only validator
	// at index 1 should be slashed and exited. We confirm this below.
	if newRegistry[1].ExitEpoch != beaconState.Validators()[1].ExitEpoch {
		t.Errorf(
			`
			Expected validator at index 1's exit epoch to match
			%d, received %d instead
			`,
			beaconState.Validators()[1].ExitEpoch,
			newRegistry[1].ExitEpoch,
		)
	}
}
