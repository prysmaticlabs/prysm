package helpers

import (
	"strconv"
	"testing"

	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func Test_ValidatorStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpb.Validator
		epoch     types.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    ethpb.ValidatorStatus
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(tt.args.validator))
			require.NoError(t, err)
			got, err := ValidatorStatus(readOnlyVal, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ValidatorSubStatus(t *testing.T) {
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch

	type args struct {
		validator *ethpb.Validator
		epoch     types.Epoch
	}
	tests := []struct {
		name    string
		args    args
		want    ethpb.ValidatorStatus
		wantErr bool
	}{
		{
			name: "pending initialized",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            farFutureEpoch,
					ActivationEligibilityEpoch: farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING_INITIALIZED,
		},
		{
			name: "pending queued",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:            10,
					ActivationEligibilityEpoch: 2,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_PENDING_QUEUED,
		},
		{
			name: "active ongoing",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       farFutureEpoch,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_ONGOING,
		},
		{
			name: "active slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         true,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_SLASHED,
		},
		{
			name: "active exiting",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch: 3,
					ExitEpoch:       30,
					Slashed:         false,
				},
				epoch: types.Epoch(5),
			},
			want: ethpb.ValidatorStatus_ACTIVE_EXITING,
		},
		{
			name: "exited slashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           true,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED_SLASHED,
		},
		{
			name: "exited unslashed",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					Slashed:           false,
				},
				epoch: types.Epoch(35),
			},
			want: ethpb.ValidatorStatus_EXITED_UNSLASHED,
		},
		{
			name: "withdrawal possible",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL_POSSIBLE,
		},
		{
			name: "withdrawal done",
			args: args{
				validator: &ethpb.Validator{
					ActivationEpoch:   3,
					ExitEpoch:         30,
					WithdrawableEpoch: 40,
					EffectiveBalance:  0,
					Slashed:           false,
				},
				epoch: types.Epoch(45),
			},
			want: ethpb.ValidatorStatus_WITHDRAWAL_DONE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(tt.args.validator))
			require.NoError(t, err)
			got, err := ValidatorSubStatus(readOnlyVal, tt.args.epoch)
			require.NoError(t, err)
			if got != tt.want {
				t.Errorf("validatorSubStatus() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// This test verifies how many validator statuses have meaningful values.
// The first expected non-meaningful value will have x.String() equal to its numeric representation.
// This test assumes we start numbering from 0 and do not skip any values.
// Having a test like this allows us to use e.g. `if value < 10` for validity checks.
func TestNumberOfStatuses(t *testing.T) {
	lastValidEnumValue := 12
	x := ethpb.ValidatorStatus(lastValidEnumValue)
	assert.NotEqual(t, strconv.Itoa(lastValidEnumValue), x.String())
	x = ethpb.ValidatorStatus(lastValidEnumValue + 1)
	assert.Equal(t, strconv.Itoa(lastValidEnumValue+1), x.String())
}
