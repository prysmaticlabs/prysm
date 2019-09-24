package trieutil

import (
"testing"
)

func TestNextPowerOf2(t *testing.T) {
	tests := []struct {
		input int
		want int
	}{
		{input:0, want:0},
		{input:1, want:1},
		{input:2, want:2},
		{input:3, want:4},
		{input:5, want:8},
		{input:9, want:16},
		{input:20, want:32},
	}
	for _, tt := range tests {
		if got := NextPowerOf2(tt.input); got != tt.want {
			t.Errorf("NextPowerOf2() = %v, want %v", got, tt.want)
		}
	}
}

func TestPrevPowerOf2(t *testing.T) {
	tests := []struct {
		input int
		want int
	}{
		{input:0, want:0},
		{input:1, want:1},
		{input:2, want:2},
		{input:3, want:2},
		{input:5, want:4},
		{input:9, want:8},
		{input:20, want:16},
	}
	for _, tt := range tests {
		if got := PrevPowerOf2(tt.input); got != tt.want {
			t.Errorf("PrevPowerOf2() = %v, want %v", got, tt.want)
		}
	}
}
