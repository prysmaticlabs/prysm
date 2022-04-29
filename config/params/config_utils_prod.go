//go:build !develop
// +build !develop

package params

import (
	"github.com/mohae/deepcopy"
)

var beaconConfig = MainnetConfig()

// BeaconConfig retrieves beacon chain config.
func BeaconConfig() *BeaconChainConfig {
	return beaconConfig
}

// OverrideBeaconConfig by replacing the config. The preferred pattern is to
// call BeaconConfig(), change the specific parameters, and then call
// OverrideBeaconConfig(c). Any subsequent calls to params.BeaconConfig() will
// return this new configuration.
func OverrideBeaconConfig(c *BeaconChainConfig) {
	c.InitializeForkSchedule()
	name, ok := reverseConfigNames[c.ConfigName]
	// if name collides with an existing config name, override it, because the fork versions probably conflict
	if !ok {
		// otherwise define it as the special "Dynamic" name, ie for a config loaded from a file at runtime
		name = Dynamic
	}
	KnownConfigs[name] = func() *BeaconChainConfig { return c }
	rebuildKnownForkVersions()
	beaconConfig = c
}

// Copy returns a copy of the config object.
func (b *BeaconChainConfig) Copy() *BeaconChainConfig {
	config, ok := deepcopy.Copy(*b).(BeaconChainConfig)
	if !ok {
		config = *beaconConfig
	}
	return &config
}
