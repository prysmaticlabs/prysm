package helpers_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestAttestationDataSlot_OK(t *testing.T) {
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	offset := uint64(0)
	committeeCount, _ := helpers.CommitteeCount(beaconState, 0)
	expect := offset / (committeeCount / params.BeaconConfig().SlotsPerEpoch)
	attSlot, err := helpers.AttestationDataSlot(beaconState, &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{
			Shard: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attSlot != expect {
		t.Errorf("Expected %d, received %d", expect, attSlot)
	}
}

func TestAttestationDataSlot_ReturnsErrorWithNilState(t *testing.T) {
	s, err := helpers.AttestationDataSlot(nil /*state*/, &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 0},
		Crosslink: &ethpb.Crosslink{
			Shard: 0,
		},
	})
	if err != helpers.ErrAttestationDataSlotNilState {
		t.Errorf("Expected an error, but received %v", err)
		t.Logf("attestation slot=%v", s)
	}
}

func TestAttestationDataSlot_ReturnsErrorWithNilData(t *testing.T) {
	s, err := helpers.AttestationDataSlot(&pb.BeaconState{}, nil /*data*/)
	if err != helpers.ErrAttestationDataSlotNilData {
		t.Errorf("Expected an error, but received %v", err)
		t.Logf("attestation slot=%v", s)
	}
}

func TestAttestationDataSlot_ReturnsErrorWithErroneousTargetEpoch(t *testing.T) {
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := helpers.AttestationDataSlot(beaconState, &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 1<<63 - 1 /* Far future epoch */},
	})
	if err == nil {
		t.Error("Expected an error, but received nil")
		t.Logf("attestation slot=%v", s)
	}
}

func TestAttestationDataSlot_ReturnsErrorWhenTargetEpochLessThanCurrentEpoch(t *testing.T) {
	deposits, _ := testutil.SetupInitialDeposits(t, 100)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &ethpb.Eth1Data{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := helpers.AttestationDataSlot(beaconState, &ethpb.AttestationData{
		Target: &ethpb.Checkpoint{Epoch: 2},
	})
	if err == nil {
		t.Error("Expected an error, but received nil")
		t.Logf("attestation slot=%v", s)
	}
}
