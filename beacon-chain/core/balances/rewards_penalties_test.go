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
		{16384 * params.BeaconConfig().MaxDepositInGwei, 0},
		{50000 * params.BeaconConfig().MaxDepositInGwei, 0},
		{100000 * params.BeaconConfig().MaxDepositInGwei, 0},
		{500000 * params.BeaconConfig().MaxDepositInGwei, 0},
		{1000000 * params.BeaconConfig().MaxDepositInGwei, 0},
		{2000000 * params.BeaconConfig().MaxDepositInGwei, 0},
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
		{params.BeaconConfig().MinOnlineDepositSize * params.BeaconConfig().Gwei, 0},
		{30 * 1e9, 0},
		{params.BeaconConfig().MaxDepositInGwei, 0},
		{40 * 1e9, 0},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{tt.a},
		}
		b := BaseReward(state, 0, params.BeaconConfig().BaseRewardQuotient)
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
		{1, 0},
		{2, 0},
		{5, 0},
		{10, 0},
		{50, 0},
	}
	for _, tt := range tests {
		state := &pb.BeaconState{
			ValidatorBalances: []uint64{params.BeaconConfig().MaxDepositInGwei},
		}
		b := InactivityPenalty(state, 0, params.BeaconConfig().BaseRewardQuotient, tt.a)
		if b != tt.b {
			t.Errorf("InactivityPenalty(%d) = %d, want = %d",
				tt.a, b, tt.b)
		}
	}
}
