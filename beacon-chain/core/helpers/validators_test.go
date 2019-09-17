package helpers

import (
	"fmt"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
		validator := &ethpb.Validator{ActivationEpoch: 10, ExitEpoch: 100}
		if IsActiveValidator(validator, test.a) != test.b {
			t.Errorf("IsActiveValidator(%d) = %v, want = %v",
				test.a, IsActiveValidator(validator, test.a), test.b)
		}
	}
}

func TestIsSlashableValidator_Active(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	activeValidator := &ethpb.Validator{
		WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	slashableValidator := IsSlashableValidator(activeValidator, 0)
	if !slashableValidator {
		t.Errorf("Expected active validator to be slashable, received false")
	}
}

func TestIsSlashableValidator_BeforeWithdrawable(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	beforeWithdrawableValidator := &ethpb.Validator{
		WithdrawableEpoch: 5,
	}

	slashableValidator := IsSlashableValidator(beforeWithdrawableValidator, 3)
	if !slashableValidator {
		t.Errorf("Expected before withdrawable validator to be slashable, received false")
	}
}

func TestIsSlashableValidator_Inactive(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	inactiveValidator := &ethpb.Validator{
		ActivationEpoch:   5,
		WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	slashableValidator := IsSlashableValidator(inactiveValidator, 2)
	if slashableValidator {
		t.Errorf("Expected inactive validator to not be slashable, received true")
	}
}

func TestIsSlashableValidator_AfterWithdrawable(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	afterWithdrawableValidator := &ethpb.Validator{
		WithdrawableEpoch: 3,
	}

	slashableValidator := IsSlashableValidator(afterWithdrawableValidator, 3)
	if slashableValidator {
		t.Errorf("Expected after withdrawable validator to not be slashable, received true")
	}
}

func TestIsSlashableValidator_SlashedWithdrawalble(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}
	slashedValidator := &ethpb.Validator{
		Slashed:           true,
		ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
		WithdrawableEpoch: 1,
	}

	slashableValidator := IsSlashableValidator(slashedValidator, 2)
	if slashableValidator {
		t.Errorf("Expected slashable validator to not be slashable, received true")
	}
}

func TestIsSlashableValidator_Slashed(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	slashedValidator2 := &ethpb.Validator{
		Slashed:           true,
		ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
		WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	slashableValidator := IsSlashableValidator(slashedValidator2, 2)
	if slashableValidator {
		t.Errorf("Expected slashable validator to not be slashable, received true")
	}
}

func TestIsSlashableValidator_InactiveSlashed(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	slashedValidator2 := &ethpb.Validator{
		Slashed:           true,
		ActivationEpoch:   4,
		ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
		WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	slashableValidator := IsSlashableValidator(slashedValidator2, 2)
	if slashableValidator {
		t.Errorf("Expected slashable validator to not be slashable, received true")
	}
}

func TestBeaconProposerIndex_OK(t *testing.T) {
	ClearAllCaches()

	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
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
			index: 254,
		},
		{
			slot:  5,
			index: 391,
		},
		{
			slot:  19,
			index: 204,
		},
		{
			slot:  30,
			index: 1051,
		},
		{
			slot:  43,
			index: 1047,
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
		validators := make([]*ethpb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState := &pb.BeaconState{
			Slot:             1,
			Validators:       validators,
			RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
			ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		}
		resultChurn, err := ValidatorChurnLimit(beaconState)
		if err != nil {
			t.Fatal(err)
		}
		if resultChurn != test.wantedChurn {
			t.Errorf("ValidatorChurnLimit(%d) = %d, want = %d",
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
		if Domain(state, tt.epoch, bytesutil.Bytes4(tt.domainType)) != tt.version {
			t.Errorf("wanted domain version: %d, got: %d", tt.version, Domain(state, tt.epoch, bytesutil.Bytes4(tt.domainType)))
		}
	}
}
