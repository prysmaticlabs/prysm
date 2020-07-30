package params

// UseE2EConfig for beacon chain services.
func UseE2EConfig() {
	beaconConfig = E2ETestConfig()

	cfg := BeaconNetworkConfig().Copy()
	cfg.ChainID = 1337   // Chain ID of eth1 dev net.
	cfg.NetworkID = 1337 // Network ID of eth1 dev net.
	OverrideBeaconNetworkConfig(cfg)
}

// E2ETestConfig retrieves the configurations made specifically for E2E testing.
// Warning: This config is only for testing, it is not meant for use outside of E2E.
func E2ETestConfig() *BeaconChainConfig {
	e2eConfig := MinimalSpecConfig()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 10 // 10 seconds so E2E has enough time to process deposits and get started.

	// Time parameters.
	e2eConfig.SecondsPerSlot = 12
	e2eConfig.SlotsPerEpoch = 6
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 4
	e2eConfig.ShardCommitteePeriod = 4
	return e2eConfig
}
