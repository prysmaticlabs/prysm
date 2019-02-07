package validators

import (
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestHasVoted(t *testing.T) {
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

	// Setting bit field to 01010101.
	pendingAttestation = &pb.Attestation{
		AggregationBitfield: []byte{85},
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

func TestInitialValidatorRegistry(t *testing.T) {
	validators := InitialValidatorRegistry()
	for idx, validator := range validators {
		if !helpers.IsActiveValidator(validator, 1) {
			t.Errorf("validator %d status is not active", idx)
		}
	}
}

func TestValidatorIdx(t *testing.T) {
	var validators []*pb.Validator
	for i := 0; i < 10; i++ {
		validators = append(validators, &pb.Validator{Pubkey: []byte{}, ExitEpoch: params.BeaconConfig().FarFutureEpoch})
	}
	if _, err := ValidatorIdx([]byte("100"), validators); err == nil {
		t.Fatalf("ValidatorIdx should have failed,  there's no validator with pubkey 100")
	}
	validators[5].Pubkey = []byte("100")
	idx, err := ValidatorIdx([]byte("100"), validators)
	if err != nil {
		t.Fatalf("call ValidatorIdx failed: %v", err)
	}
	if idx != 5 {
		t.Errorf("Incorrect validator index. Wanted 5, Got %v", idx)
	}
}

func TestEffectiveBalance(t *testing.T) {
	defaultBalance := params.BeaconConfig().MaxDeposit

	tests := []struct {
		a uint64
		b uint64
	}{
		{a: 0, b: 0},
		{a: defaultBalance - 1, b: defaultBalance - 1},
		{a: defaultBalance, b: defaultBalance},
		{a: defaultBalance + 1, b: defaultBalance},
		{a: defaultBalance * 100, b: defaultBalance},
	}
	for _, test := range tests {
		state := &pb.BeaconState{ValidatorBalances: []uint64{test.a}}
		if EffectiveBalance(state, 0) != test.b {
			t.Errorf("EffectiveBalance(%d) = %d, want = %d", test.a, EffectiveBalance(state, 0), test.b)
		}
	}
}

func TestTotalEffectiveBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		27 * 1e9, 28 * 1e9, 32 * 1e9, 40 * 1e9,
	}}

	// 27 + 28 + 32 + 32 = 119
	if TotalEffectiveBalance(state, []uint64{0, 1, 2, 3}) != 119*1e9 {
		t.Errorf("Incorrect TotalEffectiveBalance. Wanted: 119, got: %d",
			TotalEffectiveBalance(state, []uint64{0, 1, 2, 3})/1e9)
	}
}

func TestGetActiveValidator(t *testing.T) {
	inputValidators := []*pb.Validator{
		{RandaoLayers: 0},
		{RandaoLayers: 1},
		{RandaoLayers: 2},
		{RandaoLayers: 3},
		{RandaoLayers: 4},
	}

	outputValidators := []*pb.Validator{
		{RandaoLayers: 1},
		{RandaoLayers: 3},
	}

	state := &pb.BeaconState{
		ValidatorRegistry: inputValidators,
	}

	validators := ActiveValidators(state, []uint32{1, 3})

	if !reflect.DeepEqual(outputValidators, validators) {
		t.Errorf("Active validators don't match. Wanted: %v, Got: %v", outputValidators, validators)
	}
}

func TestBoundaryAttestingBalance(t *testing.T) {
	state := &pb.BeaconState{ValidatorBalances: []uint64{
		25 * 1e9, 26 * 1e9, 32 * 1e9, 33 * 1e9, 100 * 1e9,
	}}

	attestedBalances := AttestingBalance(state, []uint64{0, 1, 2, 3, 4})

	// 25 + 26 + 32 + 32 + 32 = 147
	if attestedBalances != 147*1e9 {
		t.Errorf("Incorrect attested balances. Wanted: %f, got: %d", 147*1e9, attestedBalances)
	}
}

func TestBoundaryAttesters(t *testing.T) {
	var validators []*pb.Validator

	for i := 0; i < 100; i++ {
		validators = append(validators, &pb.Validator{Pubkey: []byte{byte(i)}})
	}

	state := &pb.BeaconState{ValidatorRegistry: validators}

	boundaryAttesters := Attesters(state, []uint64{5, 2, 87, 42, 99, 0})

	expectedBoundaryAttesters := []*pb.Validator{
		{Pubkey: []byte{byte(5)}},
		{Pubkey: []byte{byte(2)}},
		{Pubkey: []byte{byte(87)}},
		{Pubkey: []byte{byte(42)}},
		{Pubkey: []byte{byte(99)}},
		{Pubkey: []byte{byte(0)}},
	}

	if !reflect.DeepEqual(expectedBoundaryAttesters, boundaryAttesters) {
		t.Errorf("Incorrect boundary attesters. Wanted: %v, got: %v", expectedBoundaryAttesters, boundaryAttesters)
	}
}

func TestBoundaryAttesterIndices(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*2)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	boundaryAttestations := []*pb.PendingAttestationRecord{
		{Data: &pb.AttestationData{}, AggregationBitfield: []byte{0x10}}, // returns indices 242
		{Data: &pb.AttestationData{}, AggregationBitfield: []byte{0xF0}}, // returns indices 237,224,2
	}

	attesterIndices, err := ValidatorIndices(state, boundaryAttestations)
	if err != nil {
		t.Fatalf("Failed to run BoundaryAttesterIndices: %v", err)
	}

	if !reflect.DeepEqual(attesterIndices, []uint64{109, 97}) {
		t.Errorf("Incorrect boundary attester indices. Wanted: %v, got: %v",
			[]uint64{109, 97}, attesterIndices)
	}
}

func TestBeaconProposerIdx(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	tests := []struct {
		slot  uint64
		index uint64
	}{
		{
			slot:  1,
			index: 511,
		},
		{
			slot:  10,
			index: 2807,
		},
		{
			slot:  19,
			index: 5122,
		},
		{
			slot:  30,
			index: 7947,
		},
		{
			slot:  39,
			index: 10262,
		},
	}

	for _, tt := range tests {
		result, err := BeaconProposerIdx(state, tt.slot)
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

func TestBeaconProposerIdx_returnsErrorWithEmptyCommittee(t *testing.T) {
	_, err := BeaconProposerIdx(&pb.BeaconState{}, 0)
	expected := "empty first committee at slot 0"
	if err.Error() != expected {
		t.Errorf("Unexpected error. got=%v want=%s", err, expected)
	}
}

func TestAttestingValidatorIndices_Ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.Validator, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              0,
	}

	prevAttestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 3,
			Shard:                6,
			ShardBlockRootHash32: []byte{'B'},
		},
		AggregationBitfield: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1},
	}

	indices, err := AttestingValidatorIndices(
		state,
		6,
		[]byte{'B'},
		nil,
		[]*pb.PendingAttestationRecord{prevAttestation})
	if err != nil {
		t.Fatalf("Could not execute AttestingValidatorIndices: %v", err)
	}

	if !reflect.DeepEqual(indices, []uint64{1141, 688}) {
		t.Errorf("Could not get incorrect validator indices. Wanted: %v, got: %v",
			[]uint64{1141, 688}, indices)
	}
}

func TestAttestingValidatorIndices_OutOfBound(t *testing.T) {
	validators := make([]*pb.Validator, params.BeaconConfig().EpochLength*9)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
		Slot:              5,
	}

	attestation := &pb.PendingAttestationRecord{
		Data: &pb.AttestationData{
			Slot:                 0,
			Shard:                1,
			ShardBlockRootHash32: []byte{'B'},
		},
		AggregationBitfield: []byte{'A'}, // 01000001 = 1,7
	}

	_, err := AttestingValidatorIndices(
		state,
		1,
		[]byte{'B'},
		[]*pb.PendingAttestationRecord{attestation},
		nil)

	// This will fail because participation bitfield is length:1, committee bitfield is length 0.
	if err == nil {
		t.Error("AttestingValidatorIndices should have failed with incorrect bitfield")
	}
}

func TestAllValidatorIndices(t *testing.T) {
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
		if !reflect.DeepEqual(AllValidatorsIndices(state), tt.indices) {
			t.Errorf("AllValidatorsIndices(%v) = %v, wanted:%v",
				tt.registries, AllValidatorsIndices(state), tt.indices)
		}
	}
}

func TestProcessDeposit_PublicKeyExistsBadWithdrawalCredentials(t *testing.T) {
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
	randaoCommitment := []byte{}

	want := "expected withdrawal credentials to match"
	if _, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestProcessDeposit_PublicKeyExistsGoodWithdrawalCredentials(t *testing.T) {
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
	pubkey := []byte{4, 5, 6}
	deposit := uint64(1000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
	)
	if err != nil {
		t.Fatalf("Process deposit failed: %v", err)
	}
	if newState.ValidatorBalances[1] != 1000 {
		t.Errorf("Expected balance at index 1 to be 1000, received %d", newState.ValidatorBalances[1])
	}
}

func TestProcessDeposit_PublicKeyDoesNotExistNoEmptyValidator(t *testing.T) {
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
	randaoCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
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

func TestProcessDeposit_PublicKeyDoesNotExistEmptyValidatorExists(t *testing.T) {
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
		Slot:              params.BeaconConfig().EpochLength,
		ValidatorBalances: balances,
		ValidatorRegistry: registry,
	}
	pubkey := []byte{7, 8, 9}
	deposit := uint64(2000)
	proofOfPossession := []byte{}
	withdrawalCredentials := []byte{1}
	randaoCommitment := []byte{}

	newState, err := ProcessDeposit(
		beaconState,
		stateutils.ValidatorIndexMap(beaconState),
		pubkey,
		deposit,
		proofOfPossession,
		withdrawalCredentials,
		randaoCommitment,
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

func TestActivateValidatorGenesis_Ok(t *testing.T) {
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

func TestActivateValidator_Ok(t *testing.T) {
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

func TestInitiateValidatorExit_Ok(t *testing.T) {
	state := &pb.BeaconState{ValidatorRegistry: []*pb.Validator{{}, {}, {}}}
	newState := InitiateValidatorExit(state, 2)
	if newState.ValidatorRegistry[0].StatusFlags != pb.Validator_INITIAL {
		t.Errorf("Wanted flag INITIAL, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
	if newState.ValidatorRegistry[2].StatusFlags != pb.Validator_INITIATED_EXIT {
		t.Errorf("Wanted flag ACTIVE_PENDING_EXIT, got %v", newState.ValidatorRegistry[0].StatusFlags)
	}
}

func TestExitValidator_Ok(t *testing.T) {
	state := &pb.BeaconState{
		Slot:                    100, // epoch 2
		LatestPenalizedBalances: []uint64{0},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch, Pubkey: []byte{'B'}},
		},
	}
	newState, err := ExitValidator(state, 0)
	if err != nil {
		t.Fatalf("could not execute ExitValidator:%v", err)
	}

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
		Slot: 1,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
		},
	}
	if _, err := ExitValidator(state, 0); err == nil {
		t.Fatal("exitValidator should have failed with exiting again")
	}
}

func TestProcessPenaltiesExits_NothingHappened(t *testing.T) {
	state := &pb.BeaconState{
		ValidatorBalances: []uint64{params.BeaconConfig().MaxDeposit},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch},
		},
	}
	if ProcessPenaltiesAndExits(state).ValidatorBalances[0] !=
		params.BeaconConfig().MaxDeposit {
		t.Errorf("wanted validator balance %d, got %d",
			params.BeaconConfig().MaxDeposit,
			ProcessPenaltiesAndExits(state).ValidatorBalances[0])
	}
}

func TestProcessPenaltiesExits_ValidatorPenalized(t *testing.T) {

	latestPenalizedExits := make([]uint64, params.BeaconConfig().LatestPenalizedExitLength)
	for i := 0; i < len(latestPenalizedExits); i++ {
		latestPenalizedExits[i] = uint64(i) * params.BeaconConfig().MaxDeposit
	}

	state := &pb.BeaconState{
		Slot:                    params.BeaconConfig().LatestPenalizedExitLength / 2 * params.BeaconConfig().EpochLength,
		LatestPenalizedBalances: latestPenalizedExits,
		ValidatorBalances:       []uint64{params.BeaconConfig().MaxDeposit, params.BeaconConfig().MaxDeposit},
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().FarFutureEpoch, RandaoLayers: 1},
		},
	}

	penalty := EffectiveBalance(state, 0) *
		EffectiveBalance(state, 0) /
		params.BeaconConfig().MaxDeposit

	newState := ProcessPenaltiesAndExits(state)
	if newState.ValidatorBalances[0] != params.BeaconConfig().MaxDeposit-penalty {
		t.Errorf("wanted validator balance %d, got %d",
			params.BeaconConfig().MaxDeposit-penalty,
			newState.ValidatorBalances[0])
	}
}

func TestEligibleToExit(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 1,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}

	state = &pb.BeaconState{
		Slot: params.BeaconConfig().MinValidatorWithdrawalEpochs,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().EntryExitDelay,
				PenalizedEpoch: 1},
		},
	}
	if eligibleToExit(state, 0) {
		t.Error("eligible to exit should be true but got false")
	}
}

func TestUpdateRegistry_NoRotation(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().EpochLength,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
			{ExitEpoch: params.BeaconConfig().EntryExitDelay},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitEpoch != params.BeaconConfig().EntryExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().EntryExitDelay, validator.ExitEpoch)
		}
	}
	if newState.ValidatorRegistryUpdateEpoch != helpers.SlotToEpoch(state.Slot) {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryUpdateEpoch)
	}
}

func TestUpdateRegistry_Activate(t *testing.T) {
	state := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().EpochLength,
		ValidatorRegistry: []*pb.Validator{
			{ExitEpoch: params.BeaconConfig().EntryExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().EntryExitDelay + 1},
			{ExitEpoch: params.BeaconConfig().EntryExitDelay,
				ActivationEpoch: 5 + params.BeaconConfig().EntryExitDelay + 1},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
		},
	}
	newState, err := UpdateRegistry(state)
	if err != nil {
		t.Fatalf("could not update validator registry:%v", err)
	}
	for i, validator := range newState.ValidatorRegistry {
		if validator.ExitEpoch != params.BeaconConfig().EntryExitDelay {
			t.Errorf("could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().EntryExitDelay, validator.ExitEpoch)
		}
	}
	if newState.ValidatorRegistryUpdateEpoch != helpers.SlotToEpoch(state.Slot) {
		t.Errorf("wanted validator registry lastet change %d, got %d",
			state.Slot, newState.ValidatorRegistryUpdateEpoch)
	}
}

func TestUpdateRegistry_Exit(t *testing.T) {
	epoch := uint64(5)
	exitEpoch := helpers.EntryExitEffectEpoch(epoch)
	state := &pb.BeaconState{
		Slot: epoch * params.BeaconConfig().EpochLength,
		ValidatorRegistry: []*pb.Validator{
			{
				ExitEpoch:   exitEpoch,
				StatusFlags: pb.Validator_INITIATED_EXIT},
			{
				ExitEpoch:   exitEpoch,
				StatusFlags: pb.Validator_INITIATED_EXIT},
		},
		ValidatorBalances: []uint64{
			params.BeaconConfig().MaxDeposit,
			params.BeaconConfig().MaxDeposit,
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

func TestMaxBalanceChurn(t *testing.T) {
	tests := []struct {
		totalBalance    uint64
		maxBalanceChurn uint64
	}{
		{totalBalance: 1e9, maxBalanceChurn: params.BeaconConfig().MaxDeposit},
		{totalBalance: params.BeaconConfig().MaxDeposit, maxBalanceChurn: 512 * 1e9},
		{totalBalance: params.BeaconConfig().MaxDeposit * 10, maxBalanceChurn: 512 * 1e10},
		{totalBalance: params.BeaconConfig().MaxDeposit * 1000, maxBalanceChurn: 512 * 1e12},
	}

	for _, tt := range tests {
		churn := maxBalanceChurn(tt.totalBalance)
		if tt.maxBalanceChurn != churn {
			t.Errorf("MaxBalanceChurn was not an expected value. Wanted: %d, got: %d",
				tt.maxBalanceChurn, churn)
		}
	}
}
