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
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestProcessProposerSlashings_UnmatchedHeaderSlots(t *testing.T) {
	testutil.ResetCache()
	beaconState, _ := testutil.DeterministicGenesisState(t, 20)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          params.BeaconConfig().SlotsPerEpoch + 1,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}
	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "mismatched header slots"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_SameHeaders(t *testing.T) {
	testutil.ResetCache()
	beaconState, _ := testutil.DeterministicGenesisState(t, 2)
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 1,
					Slot:          0,
				},
			},
		},
	}

	if err := beaconState.SetSlot(currentSlot); err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := "expected slashing headers to differ"
	_, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_ValidatorNotSlashable(t *testing.T) {
	registry := []*ethpb.Validator{
		{
			PublicKey:         []byte("key"),
			Slashed:           true,
			ActivationEpoch:   0,
			WithdrawableEpoch: 0,
		},
	}
	currentSlot := uint64(0)
	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
				},
				Signature: []byte("A"),
			},
			Header_2: &ethpb.SignedBeaconBlockHeader{
				Header: &ethpb.BeaconBlockHeader{
					ProposerIndex: 0,
					Slot:          0,
				},
				Signature: []byte("B"),
			},
		},
	}

	beaconState, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       currentSlot,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}
	want := fmt.Sprintf(
		"validator with key %#x is not slashable",
		bytesutil.ToBytes48(beaconState.Validators()[0].PublicKey),
	)

	_, err = blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessProposerSlashings_AppliesCorrectStatus(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	proposerIdx := uint64(1)

	domain, err := helpers.Domain(beaconState.Fork(), 0, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	header1 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			StateRoot:     []byte("A"),
		},
	}
	signingRoot, err := helpers.ComputeSigningRoot(header1.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header1.Signature = privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()[:]

	header2 := &ethpb.SignedBeaconBlockHeader{
		Header: &ethpb.BeaconBlockHeader{
			ProposerIndex: proposerIdx,
			Slot:          0,
			StateRoot:     []byte("B"),
		},
	}
	signingRoot, err = helpers.ComputeSigningRoot(header2.Header, domain)
	if err != nil {
		t.Errorf("Could not get signing root of beacon block header: %v", err)
	}
	header2.Signature = privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()[:]

	slashings := []*ethpb.ProposerSlashing{
		{
			Header_1: header1,
			Header_2: header2,
		},
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			ProposerSlashings: slashings,
		},
	}

	newState, err := blocks.ProcessProposerSlashings(context.Background(), beaconState, block.Body)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	newStateVals := newState.Validators()
	if newStateVals[1].ExitEpoch != beaconState.Validators()[1].ExitEpoch {
		t.Errorf("Proposer with index 1 did not correctly exit,"+"wanted slot:%d, got:%d",
			newStateVals[1].ExitEpoch, beaconState.Validators()[1].ExitEpoch)
	}
}
