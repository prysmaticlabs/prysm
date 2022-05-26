package endtoend

import (
	"testing"
)

// Run mainnet e2e config with the current release validator against latest beacon node.
func TestEndToEnd_MainnetConfig_ValidatorAtCurrentRelease(t *testing.T) {
	e2eMainnet(t, true, false).run()
}
