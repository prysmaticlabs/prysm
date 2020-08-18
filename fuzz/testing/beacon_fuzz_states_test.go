package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetBeaconFuzzState(t *testing.T) {
	_, err := GetBeaconFuzzState(1)
	require.NoError(t, err)
}
