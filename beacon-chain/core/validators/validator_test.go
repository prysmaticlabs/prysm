package validators

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted_OK(t *testing.T) {
	// Setting bit field to 11111111.
	pendingAttestation := &pb.Attestation{
		AggregationBitfield: []byte{255},
	}

	for i := 0; i < len(pendingAttestation.AggregationBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AggregationBitfield, i)
		if err != nil {
			t.Errorf("checking bit failed at index: %d with : %v", i, err)
		}
		if !voted {
			t.Error("validator voted but received didn't vote")
		}
	}

	// Setting bit field to 10101000.
	pendingAttestation = &pb.Attestation{
		AggregationBitfield: []byte{84},
	}

	for i := 0; i < len(pendingAttestation.AggregationBitfield); i++ {
		voted, err := bitutil.CheckBit(pendingAttestation.AggregationBitfield, i)
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

func TestBoundaryAttesterIndices_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}
	validators := make([]*pb.Validator, params.BeaconConfig().SlotsPerEpoch*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorRegistry: validators,
	}

	boundaryAttestations := []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: []byte{0xC0}}, // returns indices 242
		{Data: &pb.AttestationData{Slot: params.BeaconConfig().GenesisSlot},
			AggregationBitfield: []byte{0xC0}}, // returns indices 237,224,2
	}

	attesterIndices, err := ValidatorIndices(state, boundaryAttestations)
	if err != nil {
		t.Fatalf("Failed to run BoundaryAttesterIndices: %v", err)
	}

	if !reflect.DeepEqual(attesterIndices, []uint64{123, 65}) {
		t.Errorf("Incorrect boundary attester indices. Wanted: %v, got: %v",
			[]uint64{123, 65}, attesterIndices)
	}
}

func TestAttestingValidatorIndices_OK(t *testing.T) {
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

	prevAttestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                    params.BeaconConfig().GenesisSlot + 3,
			Shard:                   6,
			CrosslinkDataRootHash32: []byte{'B'},
		},
		AggregationBitfield: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1},
	}

	indices, err := AttestingValidatorIndices(
		state,
		6,
		[]byte{'B'},
		nil,
		[]*pb.PendingAttestation{prevAttestation})
	if err != nil {
		t.Fatalf("Could not execute AttestingValidatorIndices: %v", err)
	}

	if !reflect.DeepEqual(indices, []uint64{1134, 1150}) {
		t.Errorf("Could not get incorrect validator indices. Wanted: %v, got: %v",
			[]uint64{1134, 1150}, indices)
	}
}

func TestAttestingValidatorIndices_OutOfBound(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().SlotsPerEpoch*9)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              5,
	}

	attestation := &pb.PendingAttestation{
		Data: &pb.AttestationData{
			Slot:                    0,
			Shard:                   1,
			CrosslinkDataRootHash32: []byte{'B'},
		},
		AggregationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	_, err := AttestingValidatorIndices(
		state,
		1,
		[]byte{'B'},
		[]*pb.PendingAttestation{attestation},
		nil)

	// This will fail because participation bitfield is length:1, committee bitfield is length 0.
	if err == nil {
		t.Error("AttestingValidatorIndices should have failed with incorrect bitfield")
	}
}

func TestAllValidatorIndices_OK(t *testing.T) {
	tests := []struct {
		registries []*pb.Validator
		indices    []uint64
	}{
		{registries: []*pb.Validator{}, indices: []uint64{}},
		{registries: []*pb.Validator{{}}, indices: []uint64{0}},
		{registries: []*pb.Validator{{}, {}, {}, {}}, indices: []uint64{0, 1, 2, 3}},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{ValidatorRegistry: tt.registries}
		if !reflect.DeepEqual(allValidatorsIndices(state), tt.indices) {
			t.Errorf("AllValidatorsIndices(%v) = %v, wanted:%v",
				tt.registries, allValidatorsIndices(state), tt.indices)
		}
	}
}

func TestProcessDeposit_BadWithdrawalCredentials(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{0},
		},
	}
	beaconState := &pb.BeaconState{
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}

	want := "expected withdrawal credentials to match"
	if _, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestProcessDeposit_GoodWithdrawalCredentials(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{0, 0}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{2}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.ValidatorBalances[2] != 1000 {
		t.Errorf("Expected balance at index 1 to be 1000, received %d", newState.ValidatorBalances[2])
	}
}

func TestProcessDeposit_RepeatedDeposit(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey: []byte{1, 2, 3},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{0, 50}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.ValidatorBalances[1] != 1050 {
		t.Errorf("Expected balance at index 1 to be 1050, received %d", newState.ValidatorBalances[1])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExist(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1, 2, 3},
			WithdrawalCredentialsHash32: []byte{2},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{1000, 1000}
	beaconState := &pb.BeaconState{
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 3 {
		t.Errorf("Expected validator balances list to increase by 1, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[2] != 2000 {
		t.Errorf("Expected new validator have balance of %d, received %d", 2000, newState.ValidatorBalances[2])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistAndEmptyValidator(t *testing.T) {
	registry := []*pb.Validator{
		{
			Pubkey:                      []byte{1, 2, 3},
			WithdrawalCredentialsHash32: []byte{2},
		},
		{
			Pubkey:                      []byte{4, 5, 6},
			WithdrawalCredentialsHash32: []byte{1},
		},
	}
	balances := []uint64{0, 1000}
	beaconState := &pb.BeaconState{
		Slot:              params.BeaconConfig().SlotsPerEpoch,
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if len(newState.ValidatorBalances) != 3 {
		t.Errorf("Expected validator balances list to be 3, received len %d", len(newState.ValidatorBalances))
	}
	if newState.ValidatorBalances[len(newState.ValidatorBalances)-1] != 2000 {
		t.Errorf("Expected validator at last index to have balance of %d, received %d", 2000, newState.ValidatorBalances[0])
	}
}

func TestActivateValidatorGenesis_OK(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, true)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	if newState.ValidatorRegistry[0].ActivationEpoch != params.BeaconConfig().GenesisEpoch {
		t.Errorf("Wanted activation slot = genesis slot, got %d",
			newState.ValidatorRegistry[0].ActivationEpoch)
	}
}

func TestActivateValidator_OK(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 100, // epoch 2
		ValidatorRegistry: []*pb.Validator{
			{Pubkey: []byte{'A'}},
		},
	}
	newState, err := ActivateValidator(state, 0, false)
	if err != nil {
		t.Fatalf("could not execute activateValidator:%v", err)
	}
	currentEpoch := helpers.CurrentEpoch(state)
	wantedEpoch := helpers.EntryExitEffectEpoch(currentEpoch)
	if newState.ValidatorRegistry[0].ActivationEpoch != wantedEpoch {
		t.Errorf("Wanted activation slot = %d, got %d",
			wantedEpoch,
			newState.ValidatorRegistry[0].ActivationEpoch)
	}
}

func TestInitiateValidatorExit_OK(t *testing.T) {
	state := &pb.BeaconState{ValidatorRegistry: []*pb.Validator{{}, {}, {}}}
	newState := InitiateValidatorExit(state, 2)
	if newState.ValidatorRegistry[0].StatusFlags != pb.Validator_INITIAL {
		t.Errorf("Wanted flag INITIAL, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
	if newState.ValidatorRegistry[2].StatusFlags != pb.Validator_INITIATED_EXIT {
		t.Errorf("Wanted flag ACTIVE_PENDING_EXIT, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
}

func TestExitValidator_OK(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                  100, // epoch 2
		LatestSlashedBalances: []uint64{0},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch, Pubkey: []byte{'B'}},
		},
	}
	newState := ExitValidator(state, 0)

	currentEpoch := helpers.CurrentEpoch(state)
	wantedEpoch := helpers.EntryExitEffectEpoch(currentEpoch)
	if newState.ValidatorRegistry[0].ExitEpoch != wantedEpoch {
		t.Errorf("Wanted exit slot %d, got %d",
			wantedEpoch,
			newState.ValidatorRegistry[0].ExitEpoch)
	}
}

func TestExitValidator_AlreadyExited(t *testing.T) {
	state := &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisEpoch + 1000,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
		},
	}
	state = ExitValidator(state, 0)
	if state.ValidatorRegistry[0].ExitEpoch != params.BeaconConfig().ActivationExitDelay {
		t.Error("Expected exited validator to stay exited")
	}
}

func TestSlashValidator_AlreadyWithdrawn(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 100,
		ValidatorRegistry: []*pb.Validator{
			{WithdrawalEpoch: 1},
		},
	}
	want := fmt.Sprintf("withdrawn validator 0 could not get slashed, current slot: %d, withdrawn slot %d",
		state.Slot, helpers.StartSlot(state.ValidatorRegistry[0].WithdrawalEpoch))
	if _, err := SlashValidator(state, 0); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error: %s, received %v", want, err)
	}
}

func TestProcessPenaltiesExits_NothingHappened(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositAmount},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		},
	}
	if ProcessPenaltiesAndExits(state).ValidatorBalances[0] !=
		params.BeaconConfig().MaxDepositAmount {
		t.Errorf("wanted validator balance %d, got %d",
			params.BeaconConfig().MaxDepositAmount,
			ProcessPenaltiesAndExits(state).ValidatorBalances[0])
	}
}

func TestProcessPenaltiesExits_ValidatorSlashed(t *testing.T) {

	latestSlashedExits := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedExits); i++ {
		latestSlashedExits[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}

	state := &pb.BeaconState{
		Slot:                  params.BeaconConfig().LatestSlashedExitLength / 2 * params.BeaconConfig().SlotsPerEpoch,
		LatestSlashedBalances: latestSlashedExits,
		ValidatorBalances:     []uint64{params.BeaconConfig().MaxDepositAmount, params.BeaconConfig().MaxDepositAmount},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		},
	}

	penalty := helpers.EffectiveBalance(state, 0) *
		helpers.EffectiveBalance(state, 0) /
		params.BeaconConfig().MaxDepositAmount

	newState := ProcessPenaltiesAndExits(state)
	if newState.ValidatorBalances[0] != params.BeaconConfig().MaxDepositAmount-penalty {
		t.Errorf("wanted validator balance %d, got %d",
			params.BeaconConfig().MaxDepositAmount-penalty,
			newState.ValidatorBalances[0])
	}
}

func TestEligibleToExit_OK(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}

	state = &pb.BeaconState{
		Slot: params.BeaconConfig().MinValidatorWithdrawalDelay,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay,
				SlashedEpoch: 1},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}
}

func TestUpdateRegistry_NoRotation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitEpoch != params.BeaconConfig().ActivationExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().ActivationExitDelay, validator.ExitEpoch)
		}
	}
	if newState.ValidatorRegistryUpdateEpoch != helpers.SlotToEpoch(state.Slot) {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryUpdateEpoch)
	}
}

func TestUpdateRegistry_Activations(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().ActivationExitDelay + 1},
			{ExitEpoch: params.BeaconConfig().ActivationExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().ActivationExitDelay + 1},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitEpoch != params.BeaconConfig().ActivationExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().ActivationExitDelay, validator.ExitEpoch)
		}
	}
	if newState.ValidatorRegistryUpdateEpoch != helpers.SlotToEpoch(state.Slot) {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryUpdateEpoch)
	}
}

func TestUpdateRegistry_Exits(t *testing.T) {
	epoch := uint64(5)
	exitEpoch := helpers.EntryExitEffectEpoch(epoch)
	state := &pb.BeaconState{
		Slot: epoch * params.BeaconConfig().SlotsPerEpoch,
		ValidatorRegistry: []*pb.Validator{
			{
				ExitEpoch:   exitEpoch,
				StatusFlags: pb.Validator_INITIATED_EXIT},
			{
				ExitEpoch:   exitEpoch,
				StatusFlags: pb.Validator_INITIATED_EXIT},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDepositAmount,
			params.BeaconConfig().MaxDepositAmount,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitEpoch != exitEpoch {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i,
				exitEpoch,
				validator.ExitEpoch)
		}
	}
	if newState.ValidatorRegistryUpdateEpoch != helpers.SlotToEpoch(state.Slot) {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryUpdateEpoch)
	}
}

func TestMaxBalanceChurn_OK(t *testing.T) {
	maxDepositAmount := params.BeaconConfig().MaxDepositAmount
	tests := []struct {
		totalBalance    uint64
		maxBalanceChurn uint64
	}{
		{totalBalance: 1e9, maxBalanceChurn: maxDepositAmount},
		{totalBalance: maxDepositAmount, maxBalanceChurn: maxDepositAmount},
		{totalBalance: maxDepositAmount * 10, maxBalanceChurn: maxDepositAmount},
		{totalBalance: params.BeaconConfig().MaxDepositAmount * 1000, maxBalanceChurn: 5 * 1e11},
	}

	for _, tt := range tests {
		churn := maxBalanceChurn(tt.totalBalance)
		if tt.maxBalanceChurn != churn {
			t.Errorf("MaxBalanceChurn was not an expected value. Wanted: %d, got: %d",
				tt.maxBalanceChurn, churn)
		}
	}
}

func TestInitializeValidatoreStore(t *testing.T) {
	registry := make([]*pb.Validator, 0)
	indices := make([]uint64, 0)
	validatorsLimit := 100
	for i := 0; i < validatorsLimit; i++ {
		registry = append(registry, &pb.Validator{
			Pubkey:          []byte(strconv.Itoa(i)),
			ActivationEpoch: params.BeaconConfig().GenesisEpoch,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		})
		indices = append(indices, uint64(i))
	}

	bState := &pb.BeaconState{
		ValidatorRegistry: registry,
		Slot:              params.BeaconConfig().GenesisSlot,
	}

	if _, ok := vStore.activatedValidators[helpers.CurrentEpoch(bState)]; ok {
		t.Fatalf("Validator store already has indices saved in this epoch")
	}

	InitializeValidatorStore(bState)
	retrievedIndices := vStore.activatedValidators[helpers.CurrentEpoch(bState)]

	if !reflect.DeepEqual(retrievedIndices, indices) {
		t.Errorf("Saved active indices are not the same as the one in the validator store, got %v but expected %v", retrievedIndices, indices)
	}
}
