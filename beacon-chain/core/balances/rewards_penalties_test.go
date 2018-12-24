package balances

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBaseRewardQuotient(t *testing.T) {
	if params.BeaconConfig().BaseRewardQuotient != 1<<10 {
		t.Errorf("BaseRewardQuotient should be 1024 for these tests to pass")
	}
	if params.BeaconConfig().Gwei != 1e9 {
		t.Errorf("BaseRewardQuotient should be 1e9 for these tests to pass")
	}

	tests := []struct {
		a uint64
		b uint64
	}{
		{0, 0},
		{1e6 * params.BeaconConfig().Gwei, 1024000},  //1M ETH staked, 9.76% interest.
		{2e6 * params.BeaconConfig().Gwei, 1447936},  //2M ETH staked, 6.91% interest.
		{5e6 * params.BeaconConfig().Gwei, 2289664},  //5M ETH staked, 4.36% interest.
		{10e6 * params.BeaconConfig().Gwei, 3237888}, // 10M ETH staked, 3.08% interest.
		{20e6 * params.BeaconConfig().Gwei, 4579328}, // 20M ETH staked, 2.18% interest.
	}
	for _, tt := range tests {
		b := BaseRewardQuotient(tt.a)
		if b != tt.b {
			t.Errorf("BaseRewardQuotient(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}

func TestBaseReward(t *testing.T) {

	tests := []struct {
		a uint64
		b uint64
	}{
		{0, 0},
		{params.BeaconConfig().MinOnlineDepositSize * params.BeaconConfig().Gwei, 988},
		{30 * 1e9, 1853},
		{params.BeaconConfig().MaxDepositInGwei, 1976},
		{40 * 1e9, 1976},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{tt.a},
		}
		// Assume 10M Eth staked (base reward quotient: 3237888).
		b := BaseReward(state, 0, 3237888)
		if b != tt.b {
			t.Errorf("BaseReward(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}

func TestInactivityPenalty(t *testing.T) {

	tests := []struct {
		a uint64
		b uint64
	}{
		{1, 2929},
		{2, 3883},
		{5, 6744},
		{10, 11512},
		{50, 49659},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositInGwei},
		}
		// Assume 10 ETH staked (base reward quotient: 3237888).
		b := InactivityPenalty(state, 0, 3237888, tt.a)
		if b != tt.b {
			t.Errorf("InactivityPenalty(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}
