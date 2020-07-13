package params

// UseE2EConfig for beacon chain services.
func UseE2EConfig() {
	beaconConfig = E2ETestConfig()
}

// E2ETestConfig retrieves the configurations made specifically for E2E testing.
// Warning: This config is only for testing, it is not meant for use outside of E2E.
func E2ETestConfig() *BeaconChainConfig {
	e2eConfig := MinimalSpecConfig()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 30 // 30 seconds so E2E has enough time to process deposits and get started.

	// Time parameters.
	e2eConfig.SecondsPerSlot = 10
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 4
	e2eConfig.ShardCommitteePeriod = 4
	return e2eConfig
}
