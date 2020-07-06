package blocks_test

import (
	"context"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessVoluntaryExits_ValidatorNotActive(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: 0,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "non-active validator cannot exit"

	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_InvalidExitEpoch(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				Epoch: 10,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "expected current epoch >= exit epoch"

	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_NotActiveLongEnoughToExit(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
				Epoch:          0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Slot:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	want := "validator has not been active long enough to exit"
	_, err = blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessVoluntaryExits_AppliesCorrectStatus(t *testing.T) {
	exits := []*ethpb.SignedVoluntaryExit{
		{
			Exit: &ethpb.VoluntaryExit{
				ValidatorIndex: 0,
				Epoch:          0,
			},
		},
	}
	registry := []*ethpb.Validator{
		{
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
			ActivationEpoch: 0,
		},
	}
	state, err := stateTrie.InitializeFromProto(&pb.BeaconState{
		Validators: registry,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Slot: params.BeaconConfig().SlotsPerEpoch * 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = state.SetSlot(state.Slot() + (params.BeaconConfig().ShardCommitteePeriod * params.BeaconConfig().SlotsPerEpoch))
	if err != nil {
		t.Fatal(err)
	}

	priv := bls.RandKey()
	val, err := state.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	val.PublicKey = priv.PublicKey().Marshal()[:]
	if err := state.UpdateValidatorAtIndex(0, val); err != nil {
		t.Fatal(err)
	}
	domain, err := helpers.Domain(state.Fork(), helpers.CurrentEpoch(state), params.BeaconConfig().DomainVoluntaryExit, state.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	signingRoot, err := helpers.ComputeSigningRoot(exits[0].Exit, domain)
	if err != nil {
		t.Error(err)
	}
	sig := priv.Sign(signingRoot[:])
	exits[0].Signature = sig.Marshal()
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			VoluntaryExits: exits,
		},
	}

	newState, err := blocks.ProcessVoluntaryExits(context.Background(), state, block.Body)
	if err != nil {
		t.Fatalf("Could not process exits: %v", err)
	}
	newRegistry := newState.Validators()
	if newRegistry[0].ExitEpoch != helpers.ActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch) {
		t.Errorf("Expected validator exit epoch to be %d, got %d",
			helpers.ActivationExitEpoch(state.Slot()/params.BeaconConfig().SlotsPerEpoch), newRegistry[0].ExitEpoch)
	}
}
