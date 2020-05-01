package precompute_test

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestProcessSlashingsPrecompute_NotSlashed(t *testing.T) {
	s, err := beaconstate.InitializeFromProto(&pb.BeaconState{
		Slot:       0,
		Validators: []*ethpb.Validator{{Slashed: true}},
		Balances:   []uint64{params.BeaconConfig().MaxEffectiveBalance},
		Slashings:  []uint64{0, 1e9},
	})
	if err != nil {
		t.Fatal(err)
	}
	pBal := &precompute.Balance{ActiveCurrentEpoch: params.BeaconConfig().MaxEffectiveBalance}
	if err := precompute.ProcessSlashingsPrecompute(s, pBal); err != nil {
		t.Fatal(err)
	}

	wanted := params.BeaconConfig().MaxEffectiveBalance
	if s.Balances()[0] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, s.Balances()[0])
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
			ab := uint64(0)
			for i, b := range tt.state.Balances {
				// Skip validator 0 since it's slashed
				if i == 0 {
					continue
				}
				ab += b
			}
			pBal := &precompute.Balance{ActiveCurrentEpoch: ab}

			original := proto.Clone(tt.state)
			state, err := beaconstate.InitializeFromProto(tt.state)
			if err != nil {
				t.Fatal(err)
			}
			if err := precompute.ProcessSlashingsPrecompute(state, pBal); err != nil {
				t.Fatal(err)
			}

			if state.Balances()[0] != tt.want {
				t.Errorf(
					"ProcessSlashings({%v}) = newState; newState.Balances[0] = %d; wanted %d",
					original,
					state.Balances()[0],
					tt.want,
				)
			}
		})
	}
}
