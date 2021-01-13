package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetBeaconFuzzState(t *testing.T) {
	t.Skip("We'll need to generate test for new hardfork configs")
	_, err := GetBeaconFuzzState(1)
	require.NoError(t, err)
}
