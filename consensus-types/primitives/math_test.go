package types_test

import (
	"math"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
)

func TestMath_Mul64(t *testing.T) {
	type args struct {
		a uint64
		b uint64
	}
	tests := []struct {
		args args
		res  uint64
		err  bool
	}{
		{args: args{0, 1}, res: 0, err: false},
		{args: args{1 << 32, 1}, res: 1 << 32, err: false},
		{args: args{1 << 32, 100}, res: 429496729600, err: false},
		{args: args{1 << 32, 1 << 31}, res: 9223372036854775808, err: false},
		{args: args{1 << 32, 1 << 32}, res: 0, err: true},
		{args: args{1 << 62, 2}, res: 9223372036854775808, err: false},
		{args: args{1 << 62, 4}, res: 0, err: true},
		{args: args{1 << 63, 1}, res: 9223372036854775808, err: false},
		{args: args{1 << 63, 2}, res: 0, err: true},
	}
	for _, tt := range tests {
		got, err := types.Mul64(tt.args.a, tt.args.b)
		if tt.err && err == nil {
			t.Errorf("Mul64() Expected Error = %v, want error", tt.err)
			continue
		}
		if tt.res != got {
			t.Errorf("Mul64() %v, want %v", got, tt.res)
		}
	}
}

func TestMath_Div64(t *testing.T) {
	type args struct {
		a uint64
		b uint64
	}
	tests := []struct {
		args args
		res  uint64
		err  bool
	}{
		{args: args{0, 1}, res: 0, err: false},
		{args: args{0, 1}, res: 0},
		{args: args{1, 0}, res: 0, err: true},
		{args: args{1 << 32, 1 << 32}, res: 1},
		{args: args{429496729600, 1 << 32}, res: 100},
		{args: args{9223372036854775808, 1 << 32}, res: 1 << 31},
		{args: args{a: 1 << 32, b: 1 << 32}, res: 1},
		{args: args{9223372036854775808, 1 << 62}, res: 2},
		{args: args{9223372036854775808, 1 << 63}, res: 1},
	}
	for _, tt := range tests {
		got, err := types.Div64(tt.args.a, tt.args.b)
		if tt.err && err == nil {
			t.Errorf("Div64() Expected Error = %v, want error", tt.err)
			continue
		}
		if tt.res != got {
			t.Errorf("Div64() %v, want %v", got, tt.res)
		}
	}
}

func TestMath_Add64(t *testing.T) {
	type args struct {
		a uint64
		b uint64
	}
	tests := []struct {
		args args
		res  uint64
		err  bool
	}{
		{args: args{0, 1}, res: 1, err: false},
		{args: args{1 << 32, 1}, res: 4294967297, err: false},
		{args: args{1 << 32, 100}, res: 4294967396, err: false},
		{args: args{1 << 31, 1 << 31}, res: 4294967296, err: false},
		{args: args{1 << 63, 1 << 63}, res: 0, err: true},
		{args: args{1 << 63, 1}, res: 9223372036854775809, err: false},
		{args: args{math.MaxUint64, 1}, res: 0, err: true},
		{args: args{math.MaxUint64, 0}, res: math.MaxUint64, err: false},
		{args: args{1 << 63, 2}, res: 9223372036854775810, err: false},
	}
	for _, tt := range tests {
		got, err := types.Add64(tt.args.a, tt.args.b)
		if tt.err && err == nil {
			t.Errorf("Add64() Expected Error = %v, want error", tt.err)
			continue
		}
		if tt.res != got {
			t.Errorf("Add64() %v, want %v", got, tt.res)
		}
	}
}

func TestMath_Sub64(t *testing.T) {
	type args struct {
		a uint64
		b uint64
	}
	tests := []struct {
		args args
		res  uint64
		err  bool
	}{
		{args: args{1, 0}, res: 1},
		{args: args{0, 1}, res: 0, err: true},
		{args: args{1 << 32, 1}, res: 4294967295},
		{args: args{1 << 32, 100}, res: 4294967196},
		{args: args{1 << 31, 1 << 31}, res: 0},
		{args: args{1 << 63, 1 << 63}, res: 0},
		{args: args{1 << 63, 1}, res: 9223372036854775807},
		{args: args{math.MaxUint64, math.MaxUint64}, res: 0},
		{args: args{math.MaxUint64 - 1, math.MaxUint64}, res: 0, err: true},
		{args: args{math.MaxUint64, 0}, res: math.MaxUint64},
		{args: args{1 << 63, 2}, res: 9223372036854775806},
	}
	for _, tt := range tests {
		got, err := types.Sub64(tt.args.a, tt.args.b)
		if tt.err && err == nil {
			t.Errorf("Sub64() Expected Error = %v, want error", tt.err)
			continue
		}
		if tt.res != got {
			t.Errorf("Sub64() %v, want %v", got, tt.res)
		}
	}
}

func TestMath_Mod(t *testing.T) {
	type args struct {
		a uint64
		b uint64
	}
	tests := []struct {
		args args
		res  uint64
		err  bool
	}{
		{args: args{1, 0}, res: 0, err: true},
		{args: args{0, 1}, res: 0},
		{args: args{1 << 32, 1 << 32}, res: 0},
		{args: args{429496729600, 1 << 32}, res: 0},
		{args: args{9223372036854775808, 1 << 32}, res: 0},
		{args: args{1 << 32, 1 << 32}, res: 0},
		{args: args{9223372036854775808, 1 << 62}, res: 0},
		{args: args{9223372036854775808, 1 << 63}, res: 0},
		{args: args{1 << 32, 17}, res: 1},
		{args: args{1 << 32, 19}, res: (1 << 32) % 19},
		{args: args{math.MaxUint64, math.MaxUint64}, res: 0},
		{args: args{1 << 63, 2}, res: 0},
		{args: args{1<<63 + 1, 2}, res: 1},
	}
	for _, tt := range tests {
		got, err := types.Mod64(tt.args.a, tt.args.b)
		if tt.err && err == nil {
			t.Errorf("Mod64() Expected Error = %v, want error", tt.err)
			continue
		}
		if tt.res != got {
			t.Errorf("Mod64() %v, want %v", got, tt.res)
		}
	}
}
