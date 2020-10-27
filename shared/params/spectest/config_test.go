package spectest

import (
	"testing"

	types "github.com/farazdagi/prysm-shared-types"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestConfig(t *testing.T) {
	require.NoError(t, SetConfig(t, "minimal"))
	require.Equal(t, types.Slot(8), params.BeaconConfig().SlotsPerEpoch)
	require.NoError(t, SetConfig(t, "mainnet"))
	require.Equal(t, types.Slot(32), params.BeaconConfig().SlotsPerEpoch)
}
