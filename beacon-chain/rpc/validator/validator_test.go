package validator

import (
	"os"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestMain(m *testing.M) {
	// Use minimal config to reduce test setup time.
	reset := params.OverrideBeaconConfigWithReset(params.MinimalSpecConfig())
	retVal := m.Run()
	reset()
	os.Exit(retVal)
}
