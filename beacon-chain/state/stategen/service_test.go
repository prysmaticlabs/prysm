package stategen

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func Test_verifySlotsPerArchivePoint(t *testing.T) {
	type tc struct {
		input  uint64
		result bool
	}
	tests := []tc{
		{0, false},
		{1, false},
		{params.BeaconConfig().SlotsPerEpoch, true},
		{params.BeaconConfig().SlotsPerEpoch + 1, false},
		{params.BeaconConfig().SlotsPerHistoricalRoot, true},
		{params.BeaconConfig().SlotsPerHistoricalRoot + 1, false},
	}
	for _, tt := range tests {
		if got := verifySlotsPerArchivePoint(tt.input); got != tt.result {
			t.Errorf("verifySlotsPerArchivePoint(%d) = %v, want %v", tt.input, got, tt.result)
		}
	}
}
