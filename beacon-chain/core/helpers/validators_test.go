package helpers

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
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
	activeValidator := &ethpb.Validator{
		WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
	}

	slashableValidator := IsSlashableValidator(activeValidator, 0)
	if !slashableValidator {
		t.Errorf("Expected active validator to be slashable, received false")
	}
}

func TestIsSlashableValidator_BeforeWithdrawable(t *testing.T) {
	beforeWithdrawableValidator := &ethpb.Validator{
		WithdrawableEpoch: 5,
	}

	slashableValidator := IsSlashableValidator(beforeWithdrawableValidator, 3)
	if !slashableValidator {
		t.Errorf("Expected before withdrawable validator to be slashable, received false")
	}
}

func TestIsSlashableValidator_Inactive(t *testing.T) {
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
	afterWithdrawableValidator := &ethpb.Validator{
		WithdrawableEpoch: 3,
	}

	slashableValidator := IsSlashableValidator(afterWithdrawableValidator, 3)
	if slashableValidator {
		t.Errorf("Expected after withdrawable validator to not be slashable, received true")
	}
}

func TestIsSlashableValidator_SlashedWithdrawalble(t *testing.T) {
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
	params.SetupTestConfigCleanup(t)
	ClearCache()
	c := params.BeaconConfig()
	c.MinGenesisActiveValidatorCount = 16384
	params.OverrideBeaconConfig(c)
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount/8)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		Slot:        0,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		slot  uint64
		index uint64
	}{
		{
			slot:  1,
			index: 2039,
		},
		{
			slot:  5,
			index: 1895,
		},
		{
			slot:  19,
			index: 1947,
		},
		{
			slot:  30,
			index: 369,
		},
		{
			slot:  43,
			index: 464,
		},
	}

	for _, tt := range tests {
		ClearCache()
		if err := state.SetSlot(tt.slot); err != nil {
			t.Fatal(err)
		}
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

func TestComputeProposerIndex_Compatibility(t *testing.T) {
	validators := make([]*ethpb.Validator, params.BeaconConfig().MinGenesisActiveValidatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Validators:  validators,
		RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	})
	if err != nil {
		t.Fatal(err)
	}

	indices, err := ActiveValidatorIndices(state, 0)
	if err != nil {
		t.Fatal(err)
	}

	var proposerIndices []uint64
	seed, err := Seed(state, 0, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(i)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		index, err := ComputeProposerIndex(state, indices, seedWithSlotHash)
		if err != nil {
			t.Fatal(err)
		}
		proposerIndices = append(proposerIndices, index)
	}

	var wantedProposerIndices []uint64
	seed, err = Seed(state, 0, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		t.Fatal(err)
	}
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(i)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		index, err := ComputeProposerIndexWithValidators(state.Validators(), indices, seedWithSlotHash)
		if err != nil {
			t.Fatal(err)
		}
		wantedProposerIndices = append(wantedProposerIndices, index)
	}

	if !reflect.DeepEqual(wantedProposerIndices, proposerIndices) {
		t.Error("Wanted proposer indices from ComputeProposerIndexWithValidators does not match")
	}
}

func TestDelayedActivationExitEpoch_OK(t *testing.T) {
	epoch := uint64(9999)
	got := ActivationExitEpoch(epoch)
	wanted := epoch + 1 + params.BeaconConfig().MaxSeedLookahead
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
		validators := make([]*ethpb.Validator, test.validatorCount)
		for i := 0; i < len(validators); i++ {
			validators[i] = &ethpb.Validator{
				ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			}
		}

		beaconState, err := beaconstate.InitializeFromProto(&pb.BeaconState{
			Slot:        1,
			Validators:  validators,
			RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		})
		if err != nil {
			t.Fatal(err)
		}
		validatorCount, err := ActiveValidatorCount(beaconState, CurrentEpoch(beaconState))
		if err != nil {
			t.Fatal(err)
		}
		resultChurn, err := ValidatorChurnLimit(validatorCount)
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
		domainType [4]byte
		result     []byte
	}{
		{epoch: 1, domainType: bytesutil.ToBytes4(bytesutil.Bytes4(4)), result: bytesutil.ToBytes(947067381421703172, 32)},
		{epoch: 2, domainType: bytesutil.ToBytes4(bytesutil.Bytes4(4)), result: bytesutil.ToBytes(947067381421703172, 32)},
		{epoch: 2, domainType: bytesutil.ToBytes4(bytesutil.Bytes4(5)), result: bytesutil.ToBytes(947067381421703173, 32)},
		{epoch: 3, domainType: bytesutil.ToBytes4(bytesutil.Bytes4(4)), result: bytesutil.ToBytes(9369798235163459588, 32)},
		{epoch: 3, domainType: bytesutil.ToBytes4(bytesutil.Bytes4(5)), result: bytesutil.ToBytes(9369798235163459589, 32)},
	}
	for _, tt := range tests {
		domain, err := Domain(state.Fork, tt.epoch, tt.domainType, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(domain[:8], tt.result[:8]) {
			t.Errorf("wanted domain version: %d, got: %d", tt.result, domain)
		}
	}
}

// Test basic functionality of ActiveValidatorIndices without caching. This test will need to be
// rewritten when releasing some cache flag.
func TestActiveValidatorIndices(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	type args struct {
		state *pb.BeaconState
		epoch uint64
	}
	tests := []struct {
		name    string
		args    args
		want    []uint64
		wantErr bool
	}{
		{
			name: "all_active_epoch_10",
			args: args{
				state: &pb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []uint64{0, 1, 2},
		},
		{
			name: "some_active_epoch_10",
			args: args{
				state: &pb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
					},
				},
				epoch: 10,
			},
			want: []uint64{0, 1},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &pb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []uint64{0, 1, 3},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &pb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []uint64{0, 1, 3},
		},
		{
			name: "some_active_with_recent_new_epoch_10",
			args: args{
				state: &pb.BeaconState{
					RandaoMixes: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
					Validators: []*ethpb.Validator{
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       1,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
						{
							ActivationEpoch: 0,
							ExitEpoch:       farFutureEpoch,
						},
					},
				},
				epoch: 10,
			},
			want: []uint64{0, 2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := beaconstate.InitializeFromProto(tt.args.state)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ActiveValidatorIndices(s, tt.args.epoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ActiveValidatorIndices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ActiveValidatorIndices() got = %v, want %v", got, tt.want)
			}
			ClearCache()
		})
	}
}

func TestComputeProposerIndex(t *testing.T) {
	seed := bytesutil.ToBytes32([]byte("seed"))
	type args struct {
		validators []*ethpb.Validator
		indices    []uint64
		seed       [32]byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "all_active_indices",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{0, 1, 2, 3, 4},
				seed:    seed,
			},
			want: 2,
		},
		{ // Regression test for https://github.com/prysmaticlabs/prysm/issues/4259.
			name: "1_active_index",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{3},
				seed:    seed,
			},
			want: 3,
		},
		{
			name: "empty_active_indices",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{},
				seed:    seed,
			},
			wantErr: true,
		},
		{
			name: "active_indices_out_of_range",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{100},
				seed:    seed,
			},
			wantErr: true,
		},
		{
			name: "second_half_active",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{5, 6, 7, 8, 9},
				seed:    seed,
			},
			want: 7,
		},
		{
			name: "nil_validator",
			args: args{
				validators: []*ethpb.Validator{
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					nil, // Should never happen, but would cause a panic when it does happen.
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
					{EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
				},
				indices: []uint64{0, 1, 2, 3, 4},
				seed:    seed,
			},
			want: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bState := &pb.BeaconState{Validators: tt.args.validators}
			stTrie, err := beaconstate.InitializeFromProtoUnsafe(bState)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := ComputeProposerIndex(stTrie, tt.args.indices, tt.args.seed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeProposerIndex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ComputeProposerIndex() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsEligibleForActivationQueue(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		want      bool
	}{
		{"Eligible",
			&ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
			true},
		{"Incorrect activation eligibility epoch",
			&ethpb.Validator{ActivationEligibilityEpoch: 1, EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance},
			false},
		{"Not enough balance",
			&ethpb.Validator{ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch, EffectiveBalance: 1},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsEligibleForActivationQueue(tt.validator); got != tt.want {
				t.Errorf("IsEligibleForActivationQueue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsIsEligibleForActivation(t *testing.T) {
	tests := []struct {
		name      string
		validator *ethpb.Validator
		state     *pb.BeaconState
		want      bool
	}{
		{"Eligible",
			&ethpb.Validator{ActivationEligibilityEpoch: 1, ActivationEpoch: params.BeaconConfig().FarFutureEpoch},
			&pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 2}},
			true},
		{"Not yet finalized",
			&ethpb.Validator{ActivationEligibilityEpoch: 1, ActivationEpoch: params.BeaconConfig().FarFutureEpoch},
			&pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{}},
			false},
		{"Incorrect activation epoch",
			&ethpb.Validator{ActivationEligibilityEpoch: 1},
			&pb.BeaconState{FinalizedCheckpoint: &ethpb.Checkpoint{Epoch: 2}},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := beaconstate.InitializeFromProto(tt.state)
			if err != nil {
				t.Fatal(err)
			}
			if got := IsEligibleForActivation(s, tt.validator); got != tt.want {
				t.Errorf("IsEligibleForActivation() = %v, want %v", got, tt.want)
			}
		})
	}
}
