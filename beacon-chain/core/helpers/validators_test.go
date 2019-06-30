package helpers

import (
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestIsActiveValidator_OK(t *testing.T) {
	tests := []struct {
		a uint64
		b bool
	}{
		{a: 0, b: false},
		{a: 10, b: true},
		{a: 100, b: false},
		{a: 1000, b: false},
		{a: 64, b: true},
	}
	for _, test := range tests {
		validator := &pb.Validator{ActivationEpoch: 10, ExitEpoch: 100}
		if IsActiveValidator(validator, test.a) != test.b {
			t.Errorf("IsActiveValidator(%d) = %v, want = %v",
				test.a, IsActiveValidator(validator, test.a), test.b)
		}
	}
}

func TestBeaconProposerIndex_OK(t *testing.T) {
	ClearAllCaches()

	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Validators:       validators,
		Slot:             0,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	tests := []struct {
		slot  uint64
		index uint64
	}{
		{
			slot:  1,
			index: 1114,
		},
		{
			slot:  5,
			index: 1207,
		},
		{
			slot:  19,
			index: 264,
		},
		{
			slot:  30,
			index: 1421,
		},
		{
			slot:  43,
			index: 318,
		},
	}

	for _, tt := range tests {
		state.Slot = tt.slot
		result, err := BeaconProposerIndex(state)
		if err != nil {
			t.Errorf("Failed to get shard and committees at slot: %v", err)
		}

		if result != tt.index {
			t.Errorf(
				"Result index was an unexpected value. Wanted %d, got %d",
				tt.index,
				result,
			)
		}
	}
}

func TestBeaconProposerIndex_EmptyCommittee(t *testing.T) {
	ClearAllCaches()
	beaconState := &pb.BeaconState{
		Slot:             0,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	_, err := BeaconProposerIndex(beaconState)
	expected := fmt.Sprintf("empty first committee at slot %d", 0)
	if err.Error() != expected {
		t.Errorf("Unexpected error. got=%v want=%s", err, expected)
	}
}

func TestDelayedActivationExitEpoch_OK(t *testing.T) {
	epoch := uint64(9999)
	got := DelayedActivationExitEpoch(epoch)
	wanted := epoch + 1 + params.BeaconConfig().ActivationExitDelay
	if wanted != got {
		t.Errorf("Wanted: %d, received: %d", wanted, got)
	}
}

func TestChurnLimit_OK(t *testing.T) {
	tests := []struct {
		validatorCount int
		wantedChurn    uint64
	}{
		{validatorCount: 1000, wantedChurn: 4},
		{validatorCount: 100000, wantedChurn: 4},
		{validatorCount: 1000000, wantedChurn: 15 /* validatorCount/churnLimitQuotient */},
		{validatorCount: 2000000, wantedChurn: 30 /* validatorCount/churnLimitQuotient */},
	}
	for _, test := range tests {
		ClearAllCaches()
		validators := make([]*pb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &pb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState := &pb.BeaconState{
			Slot:             1,
			Validators:       validators,
			RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
			ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		}
		resultChurn, err := ChurnLimit(beaconState)
		if err != nil {
			t.Fatal(err)
		}
		if resultChurn != test.wantedChurn {
			t.Errorf("ChurnLimit(%d) = %d, want = %d",
				test.validatorCount, resultChurn, test.wantedChurn)
		}
	}
}

func TestDomain_OK(t *testing.T) {
	state := &pb.BeaconState{
		Fork: &pb.Fork{
			Epoch:           3,
			PreviousVersion: []byte{0, 0, 0, 2},
			CurrentVersion:  []byte{0, 0, 0, 3},
		},
	}
	tests := []struct {
		epoch      uint64
		domainType uint64
		version    uint64
	}{
		{epoch: 1, domainType: 4, version: 144115188075855876},
		{epoch: 2, domainType: 4, version: 144115188075855876},
		{epoch: 2, domainType: 5, version: 144115188075855877},
		{epoch: 3, domainType: 4, version: 216172782113783812},
		{epoch: 3, domainType: 5, version: 216172782113783813},
	}
	for _, tt := range tests {
		if Domain(state, tt.epoch, tt.domainType) != tt.version {
			t.Errorf("wanted domain version: %d, got: %d", tt.version, Domain(state, tt.epoch, tt.domainType))
		}
	}
}
