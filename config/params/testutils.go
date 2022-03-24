package params

import (
	"sync"
	"testing"
)

var beaconConfigLock sync.RWMutex

// SetupTestConfigCleanup preserves configurations allowing to modify them within tests without any
// restrictions, everything is restored after the test.
func SetupTestConfigCleanup(t testing.TB) {
	prevDefaultBeaconConfig := mainnetBeaconConfig.Copy()
	prevBeaconConfig := beaconConfig.Copy()
	prevNetworkCfg := networkConfig.Copy()
	t.Cleanup(func() {
		beaconConfigLock.Lock()
		mainnetBeaconConfig = prevDefaultBeaconConfig
		beaconConfig = prevBeaconConfig
		networkConfig = prevNetworkCfg
		beaconConfigLock.Unlock()
	})
}
