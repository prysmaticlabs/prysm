package validators

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted_OK(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.Attestation{
		AggregationBits: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.AggregationBits); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AggregationBits, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d with : %v", i, err)
		}
		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 10101000.
	pendingAttestation = &pb.Attestation{
		AggregationBits: []byte{84},
	}

	for i := 0; i < len(pendingAttestation.AggregationBits); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AggregationBits, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d : %v", i, err)
		}
		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestActivateValidatorGenesis_OK(t *testing.T) {
	state := &pb.BeaconState{
		Validators: []*pb.Validator{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, true)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	if newState.Validators[0].ActivationEpoch != 0 {
		t.Errorf("Wanted activation epoch = genesis epoch, got %d",
			newState.Validators[0].ActivationEpoch)
	}
	if newState.Validators[0].ActivationEligibilityEpoch != 0 {
		t.Errorf("Wanted activation eligibility epoch = genesis epoch, got %d",
			newState.Validators[0].ActivationEligibilityEpoch)
	}
}

func TestActivateValidator_OK(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 100, // epoch 2
		Validators: []*pb.Validator{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, false)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	wantedEpoch := helpers.DelayedActivationExitEpoch(currentEpoch)
	if newState.Validators[0].ActivationEpoch != wantedEpoch {
		t.Errorf("Wanted activation slot = %d, got %d",
			wantedEpoch,
			newState.Validators[0].ActivationEpoch)
	}
}

func TestInitiateValidatorExit_AlreadyExited(t *testing.T) {
	exitEpoch := uint64(199)
	state := &pb.BeaconState{Validators: []*pb.Validator{{
		ExitEpoch: exitEpoch},
	}}
	newState, err := InitiateValidatorExit(state, 0)
	if err != nil {
		t.Fatal(err)
	}
	if newState.Validators[0].ExitEpoch != exitEpoch {
		t.Errorf("Already exited, wanted exit epoch %d, got %d",
			exitEpoch, newState.Validators[0].ExitEpoch)
	}
}

func TestInitiateValidatorExit_ProperExit(t *testing.T) {
	exitedEpoch := uint64(100)
	idx := uint64(3)
	state := &pb.BeaconState{Validators: []*pb.Validator{
		{ExitEpoch: exitedEpoch},
		{ExitEpoch: exitedEpoch + 1},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
	}}
	newState, err := InitiateValidatorExit(state, idx)
	if err != nil {
		t.Fatal(err)
	}
	if newState.Validators[idx].ExitEpoch != exitedEpoch+2 {
		t.Errorf("Exit epoch was not the highest, wanted exit epoch %d, got %d",
			exitedEpoch+2, newState.Validators[idx].ExitEpoch)
	}
}

func TestInitiateValidatorExit_ChurnOverflow(t *testing.T) {
	exitedEpoch := uint64(100)
	idx := uint64(4)
	state := &pb.BeaconState{Validators: []*pb.Validator{
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2},
		{ExitEpoch: exitedEpoch + 2}, //over flow here
		{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
	}}
	newState, err := InitiateValidatorExit(state, idx)
	if err != nil {
		t.Fatal(err)
	}

	// Because of exit queue overflow,
	// validator who init exited has to wait one more epoch.
	wantedEpoch := state.Validators[0].ExitEpoch + 1

	if newState.Validators[idx].ExitEpoch != wantedEpoch {
		t.Errorf("Exit epoch did not cover overflow case, wanted exit epoch %d, got %d",
			wantedEpoch, newState.Validators[idx].ExitEpoch)
	}
}

func TestExitValidator_OK(t *testing.T) {
	state := &pb.BeaconState{
		Slot:      100, // epoch 2
		Slashings: []uint64{0},
		Validators: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch, Pubkey: []byte{'B'}},
		},
	}
	newState := ExitValidator(state, 0)

	currentEpoch := helpers.CurrentEpoch(state)
	wantedEpoch := helpers.DelayedActivationExitEpoch(currentEpoch)
	if newState.Validators[0].ExitEpoch != wantedEpoch {
		t.Errorf("Wanted exit slot %d, got %d",
			wantedEpoch,
			newState.Validators[0].ExitEpoch)
	}
}

func TestExitValidator_AlreadyExited(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1000,
		Validators: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
		},
	}
	state = ExitValidator(state, 0)
	if state.Validators[0].ExitEpoch != params.BeaconConfig().ActivationExitDelay {
		t.Error("Expected exited validator to stay exited")
	}
}

func TestSlashValidator_OK(t *testing.T) {
	registry := make([]*pb.Validator, 0)
	indices := make([]uint64, 0)
	balances := make([]uint64, 0)
	validatorsLimit := 100
	for i := 0; i < validatorsLimit; i++ {
		registry = append(registry, &pb.Validator{
			Pubkey:           []byte(strconv.Itoa(i)),
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		})
		indices = append(indices, uint64(i))
		balances = append(balances, params.BeaconConfig().MaxEffectiveBalance)
	}

	bState := &pb.BeaconState{
		Validators:       registry,
		Slot:             0,
		Slashings:        make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Balances:         balances,
	}

	slashedIdx := uint64(2)
	whistleIdx := uint64(10)

	state, err := SlashValidator(bState, slashedIdx, whistleIdx)
	if err != nil {
		t.Fatalf("Could not slash validator %v", err)
	}

	if !state.Validators[slashedIdx].Slashed {
		t.Errorf("Validator not slashed despite supposed to being slashed")
	}

	if state.Validators[slashedIdx].WithdrawableEpoch != helpers.CurrentEpoch(state)+params.BeaconConfig().EpochsPerSlashingsVector {
		t.Errorf("Withdrawable epoch not the expected value %d", state.Validators[slashedIdx].WithdrawableEpoch)
	}

	slashedBalance := state.Slashings[state.Slot%params.BeaconConfig().EpochsPerSlashingsVector]
	if slashedBalance != params.BeaconConfig().MaxEffectiveBalance {
		t.Errorf("Slashed balance isnt the expected amount: got %d but expected %d", slashedBalance, params.BeaconConfig().MaxEffectiveBalance)
	}

	proposer, err := helpers.BeaconProposerIndex(state)
	if err != nil {
		t.Errorf("Could not get proposer %v", err)
	}

	whistleblowerReward := slashedBalance / params.BeaconConfig().WhistleBlowingRewardQuotient
	proposerReward := whistleblowerReward / params.BeaconConfig().ProposerRewardQuotient

	if state.Balances[proposer] != params.BeaconConfig().MaxEffectiveBalance+proposerReward {
		t.Errorf("Did not get expected balance for proposer %d", state.Balances[proposer])
	}

	if state.Balances[whistleIdx] != params.BeaconConfig().MaxEffectiveBalance+whistleblowerReward-proposerReward {
		t.Errorf("Did not get expected balance for whistleblower %d", state.Balances[whistleIdx])
	}

	if state.Balances[slashedIdx] != params.BeaconConfig().MaxEffectiveBalance-whistleblowerReward {
		t.Errorf("Did not get expected balance for slashed validator %d", state.Balances[slashedIdx])
	}

}

func TestInitializeValidatoreStore(t *testing.T) {
	registry := make([]*pb.Validator, 0)
	indices := make([]uint64, 0)
	validatorsLimit := 100
	for i := 0; i < validatorsLimit; i++ {
		registry = append(registry, &pb.Validator{
			Pubkey:          []byte(strconv.Itoa(i)),
			ActivationEpoch: 0,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		})
		indices = append(indices, uint64(i))
	}

	bState := &pb.BeaconState{
		Validators: registry,
		Slot:       0,
	}

	if _, ok := VStore.activatedValidators[helpers.CurrentEpoch(bState)]; ok {
		t.Fatalf("Validator store already has indices saved in this epoch")
	}

	InitializeValidatorStore(bState)
	retrievedIndices := VStore.activatedValidators[helpers.CurrentEpoch(bState)]

	if !reflect.DeepEqual(retrievedIndices, indices) {
		t.Errorf("Saved active indices are not the same as the one in the validator store, got %v but expected %v", retrievedIndices, indices)
	}
}

func TestInsertActivatedIndices_Works(t *testing.T) {
	InsertActivatedIndices(100, []uint64{1, 2, 3})
	if !reflect.DeepEqual(VStore.activatedValidators[100], []uint64{1, 2, 3}) {
		t.Error("Activated validators aren't the same")
	}
	InsertActivatedIndices(100, []uint64{100})
	if !reflect.DeepEqual(VStore.activatedValidators[100], []uint64{1, 2, 3, 100}) {
		t.Error("Activated validators aren't the same")
	}
}
