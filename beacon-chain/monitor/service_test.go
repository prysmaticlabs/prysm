package monitor

import (
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestTrackedIndex(t *testing.T) {
	s := &Service{
		config: &ValidatorMonitorConfig{
			TrackedValidators: map[types.ValidatorIndex]interface{}{
				1: nil,
				2: nil,
			},
		},
	}
	require.Equal(t, s.TrackedIndex(types.ValidatorIndex(1)), true)
	require.Equal(t, s.TrackedIndex(types.ValidatorIndex(3)), false)
}
