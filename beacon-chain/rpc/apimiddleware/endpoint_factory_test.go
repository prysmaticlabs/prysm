package apimiddleware

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBeaconEndpointFactory_AllPathsRegistered(t *testing.T) {
	f := &BeaconEndpointFactory{}
	for _, p := range f.Paths() {
		_, err := f.Create(p)
		require.NoError(t, err, "failed to register %s", p)
	}
}
