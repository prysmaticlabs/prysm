package validators

import (
	"strconv"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted_OK(t *testing.T) {
	// Setting bitlist to 11111111.
	pendingAttestation := &ethpb.Attestation{
		AggregationBits: []byte{0xFF, 0x01},
	}

	for i := uint64(0); i < pendingAttestation.AggregationBits.Len(); i++ {
		if !pendingAttestation.AggregationBits.BitAt(i) {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 10101010.
	pendingAttestation = &ethpb.Attestation{
		AggregationBits: []byte{0xAA, 0x1},
	}

	for i := uint64(0); i < pendingAttestation.AggregationBits.Len(); i++ {
		voted := pendingAttestation.AggregationBits.BitAt(i)
		if i%2 == 0 && voted {
			t.Error("validator didn't vote but received voted")
		}
		if i%2 == 1 && !voted {
			t.Error("validator voted but received didn't vote")
		}
	}
}

func TestInitiateValidatorExit_AlreadyExited(t *testing.T) {
	exitEpoch := uint64(199)
	state := &pb.BeaconState{Validators: []*ethpb.Validator{{
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
	state := &pb.BeaconState{Validators: []*ethpb.Validator{
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
	state := &pb.BeaconState{Validators: []*ethpb.Validator{
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

func TestSlashValidator_OK(t *testing.T) {
	registry := make([]*ethpb.Validator, 0)
	balances := make([]uint64, 0)
	validatorsLimit := 100
	for i := 0; i < validatorsLimit; i++ {
		registry = append(registry, &ethpb.Validator{
			PublicKey:        []byte(strconv.Itoa(i)),
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		})
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

	whistleblowerReward := slashedBalance / params.BeaconConfig().WhistleBlowerRewardQuotient
	proposerReward := whistleblowerReward / params.BeaconConfig().ProposerRewardQuotient

	if state.Balances[proposer] != params.BeaconConfig().MaxEffectiveBalance+proposerReward {
		t.Errorf("Did not get expected balance for proposer %d", state.Balances[proposer])
	}

	if state.Balances[whistleIdx] != params.BeaconConfig().MaxEffectiveBalance+whistleblowerReward-proposerReward {
		t.Errorf("Did not get expected balance for whistleblower %d", state.Balances[whistleIdx])
	}

	if state.Balances[slashedIdx] != params.BeaconConfig().MaxEffectiveBalance-(state.Validators[slashedIdx].EffectiveBalance/params.BeaconConfig().MinSlashingPenaltyQuotient) {
		t.Errorf("Did not get expected balance for slashed validator, wanted %d but got %d",
			state.Validators[slashedIdx].EffectiveBalance/params.BeaconConfig().MinSlashingPenaltyQuotient, state.Balances[slashedIdx])
	}

}
