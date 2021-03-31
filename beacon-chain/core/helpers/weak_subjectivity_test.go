package helpers

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestWeakSubjectivityCheckptEpoch(t *testing.T) {
	tests := []struct {
		valCount uint64
		want     types.Epoch
	}{
		// Verifying these numbers aligned with the reference table defined:
		// https://github.com/ethereum/eth2.0-specs/blob/weak-subjectivity-guide/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount, want: 460},
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount * 2, want: 665},
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount * 4, want: 1075},
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount * 8, want: 1894},
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount * 16, want: 3532},
		{valCount: params.BeaconConfig().MinGenesisActiveValidatorCount * 32, want: 3532},
	}
	for _, tt := range tests {
		got, err := WeakSubjectivityCheckptEpoch(tt.valCount)
		require.NoError(t, err)
		if got != tt.want {
			t.Errorf("WeakSubjectivityCheckptEpoch() = %v, want %v", got, tt.want)
		}
	}
}
