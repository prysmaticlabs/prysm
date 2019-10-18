package precompute

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessSlashingsPrecompute_NotSlashed(t *testing.T) {
	s := &pb.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{{Slashed: true}},
		Balances:   []uint64{params.BeaconConfig().MaxEffectiveBalance},
		Slashings:  []uint64{0, 1e9},
	}
	bp := &Balance{CurrentEpoch: params.BeaconConfig().MaxEffectiveBalance}
	newState := ProcessSlashingsPrecompute(s, bp)
	wanted := params.BeaconConfig().MaxEffectiveBalance
	if newState.Balances[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Balances[0])
	}
}

func TestProcessSlashingsPrecompute_SlashedLess(t *testing.T) {

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
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
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
				Balances:  []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
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
			helpers.ClearAllCaches()

			ab := uint64(0)
			for i, b := range tt.state.Balances {
				// Skip validator 0 since it's slashed
				if i == 0 {
					continue
				}
				ab += b
			}
			bp := &Balance{CurrentEpoch: ab}

			original := proto.Clone(tt.state)
			newState := ProcessSlashingsPrecompute(tt.state, bp)

			if newState.Balances[0] != tt.want {
				t.Errorf(
					"ProcessSlashings({%v}) = newState; newState.Balances[0] = %d; wanted %d",
					original,
					newState.Balances[0],
					tt.want,
				)
			}
		})
	}
}
