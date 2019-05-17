package helpers

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestTotalBalance_OK(t *testing.T) {
	state := &pb.BeaconState{ValidatorRegistry: []*pb.Validator{
		{EffectiveBalance: 27 * 1e9}, {EffectiveBalance: 28 * 1e9},
		{EffectiveBalance: 32 * 1e9}, {EffectiveBalance: 40 * 1e9},
	}}

	if TotalBalance(state, []uint64{0, 1, 2, 3}) != 127*1e9 {
		t.Errorf("Incorrect TotalEffectiveBalance. Wanted: 127, got: %d",
			TotalBalance(state, []uint64{0, 1, 2, 3})/1e9)
	}
}

func TestGetBalance_OK(t *testing.T) {
	tests := []struct {
		i uint64
		b []uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}},
		{i: 0, b: []uint64{0, 0, 0}},
		{i: 2, b: []uint64{0, 0, 0}},
	}
	for _, test := range tests {
		state := &pb.BeaconState{Balances: test.b}
		if state.Balances[test.i] != test.b[test.i] {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.b[test.i], state.Balances[test.i])
		}
	}
}

func TestIncreseBalance_OK(t *testing.T) {
	tests := []struct {
		i  uint64
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 27*1e9 + 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 33 * 1e9, eb: 65 * 1e9},
	}
	for _, test := range tests {
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		}
		state = IncreaseBalance(state, test.i, test.nb)
		if state.Balances[test.i] != test.eb {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.eb, state.Balances[test.i])
		}
	}
}
func TestDecreseBalance_OK(t *testing.T) {
	tests := []struct {
		i  uint64
		b  []uint64
		nb uint64
		eb uint64
	}{
		{i: 0, b: []uint64{2, 28 * 1e9, 32 * 1e9}, nb: 1, eb: 1},
		{i: 1, b: []uint64{27 * 1e9, 28 * 1e9, 32 * 1e9}, nb: 0, eb: 28 * 1e9},
		{i: 2, b: []uint64{27 * 1e9, 28 * 1e9, 1}, nb: 2, eb: 0},
	}
	for _, test := range tests {
		state := &pb.BeaconState{
			ValidatorRegistry: []*pb.Validator{
				{EffectiveBalance: 4}, {EffectiveBalance: 4}, {EffectiveBalance: 4}},
			Balances: test.b,
		}
		state = DecreaseBalance(state, test.i, test.nb)
		if state.Balances[test.i] != test.eb {
			t.Errorf("Incorrect Validator balance. Wanted: %d, got: %d", test.eb, state.Balances[test.i])
		}
	}
}
