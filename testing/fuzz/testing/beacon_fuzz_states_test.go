package testing

import (
	"testing"

	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestGetBeaconFuzzState(t *testing.T) {
	_, err := BeaconFuzzState(1)
	require.NoError(t, err)
}
