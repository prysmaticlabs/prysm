package epoch_test

import (
	"bytes"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestUnslashedAttestingIndices_CanSortAndFilter(t *testing.T) {
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{Epoch: 0},
			},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators and state for the 2 attestations.
	validatorCount := 1000
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	base := &pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	indices, err := epoch.UnslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices)-1; i++ {
		if indices[i] >= indices[i+1] {
			t.Error("sorted indices not sorted or duplicated")
		}
	}

	// Verify the slashed validator is filtered.
	slashedValidator := indices[0]
	validators = state.Validators()
	validators[slashedValidator].Slashed = true
	if err = state.SetValidators(validators); err != nil {
		t.Fatal(err)
	}
	indices, err = epoch.UnslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(indices); i++ {
		if indices[i] == slashedValidator {
			t.Errorf("Slashed validator %d is not filtered", slashedValidator)
		}
	}
}

func TestUnslashedAttestingIndices_DuplicatedAttestations(t *testing.T) {
	// Generate 5 of the same attestations.
	atts := make([]*pb.PendingAttestation, 5)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{Source: &ethpb.Checkpoint{},
				Target: &ethpb.Checkpoint{Epoch: 0}},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF},
		}
	}

	// Generate validators and state for the 5 attestations.
	validatorCount := 1000
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	base := &pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	indices, err := epoch.UnslashedAttestingIndices(state, atts)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < len(indices)-1; i++ {
		if indices[i] >= indices[i+1] {
			t.Error("sorted indices not sorted or duplicated")
		}
	}
}

func TestAttestingBalance_CorrectBalance(t *testing.T) {
	helpers.ClearCache()
	// Generate 2 attestations.
	atts := make([]*pb.PendingAttestation, 2)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
				Slot:   uint64(i),
			},
			AggregationBits: bitfield.Bitlist{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01},
		}
	}

	// Generate validators with balances and state for the 2 attestations.
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	balances := make([]uint64, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	base := &pb.BeaconState{
		Slot:        2,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),

		Validators: validators,
		Balances:   balances,
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := epoch.AttestingBalance(state, atts)
	if err != nil {
		t.Fatal(err)
	}
	wanted := 256 * params.BeaconConfig().MaxEffectiveBalance
	if balance != wanted {
		t.Errorf("wanted balance: %d, got: %d", wanted, balance)
	}
}

func TestBaseReward_AccurateRewards(t *testing.T) {
	tests := []struct {
		a uint64
		b uint64
		c uint64
	}{
		{params.BeaconConfig().MinDepositAmount, params.BeaconConfig().MinDepositAmount, 505976},
		{30 * 1e9, 30 * 1e9, 2771282},
		{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance, 2862174},
		{40 * 1e9, params.BeaconConfig().MaxEffectiveBalance, 2862174},
	}
	for _, tt := range tests {
		base := &pb.BeaconState{
			Validators: []*ethpb.Validator{
				{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: tt.b}},
			Balances: []uint64{tt.a},
		}
		state, err := state.InitializeFromProto(base)
		if err != nil {
			t.Fatal(err)
		}
		c, err := epoch.BaseReward(state, 0)
		if err != nil {
			t.Fatal(err)
		}
		if c != tt.c {
			t.Errorf("epoch.BaseReward(%d) = %d, want = %d",
				tt.a, c, tt.c)
		}
	}
}

func TestProcessSlashings_NotSlashed(t *testing.T) {
	base := &pb.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{{Slashed: true}},
		Balances:   []uint64{params.BeaconConfig().MaxEffectiveBalance},
		Slashings:  []uint64{0, 1e9},
	}
	s, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := epoch.ProcessSlashings(s)
	if err != nil {
		t.Fatal(err)
	}
	wanted := params.BeaconConfig().MaxEffectiveBalance
	if newState.Balances()[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances()[0])
	}
}

func TestProcessSlashings_SlashedLess(t *testing.T) {
	tests := []struct {
		state *pb.BeaconState
		want  uint64
	}{
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance}},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 3000000000 = (32 * 1e9)        / (1 * 1e9) * (3*1e9)             / (32*1e9)      * (1 * 1e9)
			want: uint64(29000000000), // 32 * 1e9 - 3000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 1000000000 = (32 * 1e9)        / (1 * 1e9) * (3*1e9)             / (64*1e9)      * (1 * 1e9)
			want: uint64(31000000000), // 32 * 1e9 - 1000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
				Slashings: []uint64{0, 2 * 1e9},
			},
			// penalty    = validator balance / increment * (3*total_penalties) / total_balance * increment
			// 3000000000 = (32 * 1e9)        / (1 * 1e9) * (3*2e9)             / (64*1e9)      * (1 * 1e9)
			want: uint64(29000000000), // 32 * 1e9 - 3000000000
		},
		{
			state: &pb.BeaconState{
				Validators: []*ethpb.Validator{
					{Slashed: true,
						WithdrawableEpoch: params.BeaconConfig().EpochsPerSlashingsVector / 2,
						EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement},
					{ExitEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement}},
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement, params.BeaconConfig().MaxEffectiveBalance - params.BeaconConfig().EffectiveBalanceIncrement},
				Slashings: []uint64{0, 1e9},
			},
			// penalty    = validator balance           / increment * (3*total_penalties) / total_balance        * increment
			// 3000000000 = (32  * 1e9 - 1*1e9)         / (1 * 1e9) * (3*1e9)             / (31*1e9)             * (1 * 1e9)
			want: uint64(28000000000), // 31 * 1e9 - 3000000000
		},
	}

	for i, tt := range tests {
		t.Run(string(i), func(t *testing.T) {
			original := proto.Clone(tt.state)
			s, err := state.InitializeFromProto(tt.state)
			if err != nil {
				t.Fatal(err)
			}
			newState, err := epoch.ProcessSlashings(s)
			if err != nil {
				t.Fatal(err)
			}

			if newState.Balances()[0] != tt.want {
				t.Errorf(
					"ProcessSlashings({%v}) = newState; newState.Balances[0] = %d; wanted %d",
					original,
					newState.Balances()[0],
					tt.want,
				)
			}
		})
	}
}

func TestProcessFinalUpdates_CanProcess(t *testing.T) {
	s := buildState(params.BeaconConfig().SlotsPerHistoricalRoot-1, params.BeaconConfig().SlotsPerEpoch)
	ce := helpers.CurrentEpoch(s)
	ne := ce + 1
	if err := s.SetEth1DataVotes([]*ethpb.Eth1Data{}); err != nil {
		t.Fatal(err)
	}
	balances := s.Balances()
	balances[0] = 31.75 * 1e9
	balances[1] = 31.74 * 1e9
	if err := s.SetBalances(balances); err != nil {
		t.Fatal(err)
	}

	slashings := s.Slashings()
	slashings[ce] = 0
	if err := s.SetSlashings(slashings); err != nil {
		t.Fatal(err)
	}
	mixes := s.RandaoMixes()
	mixes[ce] = []byte{'A'}
	if err := s.SetRandaoMixes(mixes); err != nil {
		t.Fatal(err)
	}
	newS, err := epoch.ProcessFinalUpdates(s)
	if err != nil {
		t.Fatal(err)
	}

	// Verify effective balance is correctly updated.
	if newS.Validators()[0].EffectiveBalance != params.BeaconConfig().MaxEffectiveBalance {
		t.Errorf("effective balance incorrectly updated, got %d", s.Validators()[0].EffectiveBalance)
	}
	if newS.Validators()[1].EffectiveBalance != 31*1e9 {
		t.Errorf("effective balance incorrectly updated, got %d", s.Validators()[1].EffectiveBalance)
	}

	// Verify slashed balances correctly updated.
	if newS.Slashings()[ce] != newS.Slashings()[ne] {
		t.Errorf("wanted slashed balance %d, got %d",
			newS.Slashings()[ce],
			newS.Slashings()[ne])
	}

	// Verify randao is correctly updated in the right position.
	if mix, err := newS.RandaoMixAtIndex(ne); err != nil || bytes.Equal(mix, params.BeaconConfig().ZeroHash[:]) {
		t.Error("latest RANDAO still zero hashes")
	}

	// Verify historical root accumulator was appended.
	if len(newS.HistoricalRoots()) != 1 {
		t.Errorf("wanted slashed balance %d, got %d", 1, len(newS.HistoricalRoots()[ce]))
	}

	if newS.CurrentEpochAttestations() == nil {
		t.Error("nil value stored in current epoch attestations instead of empty slice")
	}
}

func TestProcessRegistryUpdates_NoRotation(t *testing.T) {
	base := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead},
		},
		Balances: []uint64{
			params.BeaconConfig().MaxEffectiveBalance,
			params.BeaconConfig().MaxEffectiveBalance,
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := epoch.ProcessRegistryUpdates(state)
	if err != nil {
		t.Fatal(err)
	}
	for i, validator := range newState.Validators() {
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookahead {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookahead, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_EligibleToActivate(t *testing.T) {
	base := &pb.BeaconState{
		Slot:                5 * params.BeaconConfig().SlotsPerEpoch,
		FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 6},
	}
	limit, err := helpers.ValidatorChurnLimit(0)
	if err != nil {
		t.Error(err)
	}
	for i := uint64(0); i < limit+10; i++ {
		base.Validators = append(base.Validators, &ethpb.Validator{
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ActivationEpoch:            params.BeaconConfig().FarFutureEpoch,
		})
	}
	state, err := state.InitializeFromProto(base)
	currentEpoch := helpers.CurrentEpoch(state)
	newState, err := epoch.ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators() {
		if validator.ActivationEligibilityEpoch != currentEpoch+1 {
			t.Errorf("Could not update registry %d, wanted activation eligibility epoch %d got %d",
				i, currentEpoch, validator.ActivationEligibilityEpoch)
		}
		if uint64(i) < limit && validator.ActivationEpoch != helpers.ActivationExitEpoch(currentEpoch) {
			t.Errorf("Could not update registry %d, validators failed to activate: wanted activation epoch %d, got %d",
				i, helpers.ActivationExitEpoch(currentEpoch), validator.ActivationEpoch)
		}
		if uint64(i) >= limit && validator.ActivationEpoch != params.BeaconConfig().FarFutureEpoch {
			t.Errorf("Could not update registry %d, validators should not have been activated, wanted activation epoch: %d, got %d",
				i, params.BeaconConfig().FarFutureEpoch, validator.ActivationEpoch)
		}
	}
}

func TestProcessRegistryUpdates_ActivationCompletes(t *testing.T) {
	base := &pb.BeaconState{
		Slot: 5 * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookahead + 1},
			{ExitEpoch: params.BeaconConfig().MaxSeedLookahead,
				ActivationEpoch: 5 + params.BeaconConfig().MaxSeedLookahead + 1},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := epoch.ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators() {
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookahead {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookahead, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_ValidatorsEjected(t *testing.T) {
	base := &pb.BeaconState{
		Slot: 0,
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().EjectionBalance - 1,
			},
			{
				ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
				EffectiveBalance: params.BeaconConfig().EjectionBalance - 1,
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := epoch.ProcessRegistryUpdates(state)
	if err != nil {
		t.Error(err)
	}
	for i, validator := range newState.Validators() {
		if validator.ExitEpoch != params.BeaconConfig().MaxSeedLookahead+1 {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i, params.BeaconConfig().MaxSeedLookahead+1, validator.ExitEpoch)
		}
	}
}

func TestProcessRegistryUpdates_CanExits(t *testing.T) {
	e := uint64(5)
	exitEpoch := helpers.ActivationExitEpoch(e)
	minWithdrawalDelay := params.BeaconConfig().MinValidatorWithdrawabilityDelay
	base := &pb.BeaconState{
		Slot: e * params.BeaconConfig().SlotsPerEpoch,
		Validators: []*ethpb.Validator{
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
			{
				ExitEpoch:         exitEpoch,
				WithdrawableEpoch: exitEpoch + minWithdrawalDelay},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{},
	}
	state, err := state.InitializeFromProto(base)
	if err != nil {
		t.Fatal(err)
	}
	newState, err := epoch.ProcessRegistryUpdates(state)
	if err != nil {
		t.Fatal(err)
	}
	for i, validator := range newState.Validators() {
		if validator.ExitEpoch != exitEpoch {
			t.Errorf("Could not update registry %d, wanted exit slot %d got %d",
				i,
				exitEpoch,
				validator.ExitEpoch,
			)
		}
	}
}

func buildState(slot uint64, validatorCount uint64) *state.BeaconState {
	validators := make([]*ethpb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	latestActiveIndexRoots := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestActiveIndexRoots); i++ {
		latestActiveIndexRoots[i] = params.BeaconConfig().ZeroHash[:]
	}
	latestRandaoMixes := make(
		[][]byte,
		params.BeaconConfig().EpochsPerHistoricalVector,
	)
	for i := 0; i < len(latestRandaoMixes); i++ {
		latestRandaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}
	s := testutil.NewBeaconState()
	if err := s.SetSlot(slot); err != nil {
		panic(err)
	}
	if err := s.SetBalances(validatorBalances); err != nil {
		panic(err)
	}
	if err := s.SetValidators(validators); err != nil {
		panic(err)
	}

	return s
}
