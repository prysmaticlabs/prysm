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
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              params.BeaconConfig().GenesisSlot,
	}

	tests := []struct {
		slot  uint64
		index uint64
	}{
		{
			slot:  params.BeaconConfig().GenesisSlot + 1,
			index: 504,
		},
		{
			slot:  params.BeaconConfig().GenesisSlot + 10,
			index: 2821,
		},
		{
			slot:  params.BeaconConfig().GenesisSlot + 19,
			index: 5132,
		},
		{
			slot:  params.BeaconConfig().GenesisSlot + 30,
			index: 7961,
		},
		{
			slot:  params.BeaconConfig().GenesisSlot + 39,
			index: 10272,
		},
	}

	for _, tt := range tests {
		result, err := BeaconProposerIndex(state, tt.slot)
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
	_, err := BeaconProposerIndex(&pb.BeaconState{Slot: params.BeaconConfig().GenesisSlot}, params.BeaconConfig().GenesisSlot)
	expected := fmt.Sprintf("empty first committee at slot %d", 0)
	if err.Error() != expected {
		t.Errorf("Unexpected error. got=%v want=%s", err, expected)
	}
}

func TestEntryExitEffectEpoch_OK(t *testing.T) {
	epoch := uint64(9999)
	got := EntryExitEffectEpoch(epoch)
	wanted := epoch + 1 + params.BeaconConfig().ActivationExitDelay
	if wanted != got {
		t.Errorf("Wanted: %d, received: %d", wanted, got)
	}
}
