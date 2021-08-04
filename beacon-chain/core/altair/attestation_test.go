package altair_test

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestValidatorFlag_Has(t *testing.T) {
	tests := []struct {
		name     string
		set      uint8
		expected []uint8
	}{
		{name: "none",
			set:      0,
			expected: []uint8{},
		},
		{
			name:     "source",
			set:      1,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex},
		},
		{
			name:     "target",
			set:      2,
			expected: []uint8{params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "head",
			set:      4,
			expected: []uint8{params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:     "source, target",
			set:      3,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "source, head",
			set:      5,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:     "target, head",
			set:      6,
			expected: []uint8{params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
		},
		{
			name:     "source, target, head",
			set:      7,
			expected: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, f := range tt.expected {
				require.Equal(t, true, altair.HasValidatorFlag(tt.set, f))
			}
		})
	}
}

func TestValidatorFlag_Add(t *testing.T) {
	tests := []struct {
		name          string
		set           []uint8
		expectedTrue  []uint8
		expectedFalse []uint8
	}{
		{name: "none",
			set:           []uint8{},
			expectedTrue:  []uint8{},
			expectedFalse: []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex},
			expectedFalse: []uint8{params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source, target",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex},
			expectedFalse: []uint8{params.BeaconConfig().TimelyHeadFlagIndex},
		},
		{
			name:          "source, target, head",
			set:           []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
			expectedTrue:  []uint8{params.BeaconConfig().TimelySourceFlagIndex, params.BeaconConfig().TimelyTargetFlagIndex, params.BeaconConfig().TimelyHeadFlagIndex},
			expectedFalse: []uint8{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := uint8(0)
			for _, f := range tt.set {
				b = altair.AddValidatorFlag(b, f)
			}
			for _, f := range tt.expectedFalse {
				require.Equal(t, false, altair.HasValidatorFlag(b, f))
			}
			for _, f := range tt.expectedTrue {
				require.Equal(t, true, altair.HasValidatorFlag(b, f))
			}
		})
	}
}
