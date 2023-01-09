package endtoend

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/prysmaticlabs/prysm/v3/testing/endtoend/types"
)

// Run mainnet e2e config with the current release validator against latest beacon node.
func TestEndToEnd_MainnetConfig_ValidatorAtCurrentRelease(t *testing.T) {
	r := e2eMainnet(t, true, false, types.StartAt(version.Phase0, params.E2EMainnetTestConfig()))
	r.run()
}

func TestEndToEnd_MainnetConfig_MultiClient(t *testing.T) {
	e2eMainnet(t, false, true, types.StartAt(version.Phase0, params.E2EMainnetTestConfig()), types.WithValidatorCrossClient()).run()
}
