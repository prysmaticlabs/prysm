package params

import "testing"

// SetupTestConfigCleanup preserves configurations allowing to modify them within tests without any
// restrictions, everything is restored after the test.
func SetupTestConfigCleanup(t testing.TB) {
	prevDefaultBeaconConfig := mainnetBeaconConfig.Copy()
	prevBeaconConfig := beaconConfig.Copy()
	prevNetworkCfg := mainnetNetworkConfig.Copy()
	t.Cleanup(func() {
		mainnetBeaconConfig = prevDefaultBeaconConfig
		beaconConfig = prevBeaconConfig
		mainnetNetworkConfig = prevNetworkCfg
	})
}
