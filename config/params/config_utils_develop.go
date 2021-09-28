// +build develop

package params

import (
	"sync"

	"github.com/mohae/deepcopy"
)

var beaconConfig = MainnetConfig()
var beaconConfigLock sync.RWMutex

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	beaconConfigLock.RLock()
	defer beaconConfigLock.RUnlock()
	return beaconConfig
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	beaconConfigLock.Lock()
	defer beaconConfigLock.Unlock()
	beaconConfig = c
}

// Copy returns a copy of the config object.
func (b *BeaconChainConfig) Copy() *BeaconChainConfig {
	beaconConfigLock.RLock()
	defer beaconConfigLock.RUnlock()
	config, ok := deepcopy.Copy(*b).(BeaconChainConfig)
	if !ok {
		config = *beaconConfig
	}
	return &config
}
