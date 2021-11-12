package monitor

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestTrackedIndex(t *testing.T) {
	s := &Service{
		config: &ValidatorMonitorConfig{
			TrackedValidators: []types.ValidatorIndex{types.ValidatorIndex(1), types.ValidatorIndex(2)},
		},
	}
	require.Equal(t, s.TrackedIndex(types.ValidatorIndex(1)), true)
	require.Equal(t, s.TrackedIndex(types.ValidatorIndex(3)), false)
}
