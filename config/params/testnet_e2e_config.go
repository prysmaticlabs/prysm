package params

const (
	altairE2EForkEpoch    = 6
	bellatrixE2EForkEpoch = 8
)

// E2ETestConfig retrieves the configurations made specifically for E2E testing.
//
// WARNING: This config is only for testing, it is not meant for use outside of E2E.
func E2ETestConfig() *BeaconChainConfig {
	e2eConfig := MinimalSpecConfig()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 10 // 10 seconds so E2E has enough time to process deposits and get started.
	e2eConfig.ChurnLimitQuotient = 65536

	// Time parameters.
	e2eConfig.SecondsPerSlot = 10
	e2eConfig.SlotsPerEpoch = 6
	e2eConfig.SqrRootSlotsPerEpoch = 2
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 8
	e2eConfig.EpochsPerEth1VotingPeriod = 2
	e2eConfig.ShardCommitteePeriod = 4
	e2eConfig.MaxSeedLookahead = 1

	// PoW parameters.
	e2eConfig.DepositChainID = 1337   // Chain ID of eth1 dev net.
	e2eConfig.DepositNetworkID = 1337 // Network ID of eth1 dev net.

	// Fork Parameters.
	e2eConfig.AltairForkEpoch = altairE2EForkEpoch
	e2eConfig.BellatrixForkEpoch = bellatrixE2EForkEpoch

	// Terminal Total Difficulty.
	e2eConfig.TerminalTotalDifficulty = "616"

	// Prysm constants.
	e2eConfig.ConfigName = EndToEndName
	e2eConfig.GenesisForkVersion = []byte{0, 0, 0, 253}
	e2eConfig.AltairForkVersion = []byte{1, 0, 0, 253}
	e2eConfig.BellatrixForkVersion = []byte{2, 0, 0, 253}
	e2eConfig.ShardingForkVersion = []byte{3, 0, 0, 253}

	e2eConfig.InitializeForkSchedule()
	return e2eConfig
}

func E2EMainnetTestConfig() *BeaconChainConfig {
	e2eConfig := MainnetConfig().Copy()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 25 // 25 seconds so E2E has enough time to process deposits and get started.
	e2eConfig.ChurnLimitQuotient = 65536

	// Time parameters.
	e2eConfig.SecondsPerSlot = 6
	e2eConfig.SqrRootSlotsPerEpoch = 5
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 8
	e2eConfig.ShardCommitteePeriod = 4

	// PoW parameters.
	e2eConfig.DepositChainID = 1337   // Chain ID of eth1 dev net.
	e2eConfig.DepositNetworkID = 1337 // Network ID of eth1 dev net.

	// Altair Fork Parameters.
	e2eConfig.AltairForkEpoch = altairE2EForkEpoch
	e2eConfig.BellatrixForkEpoch = bellatrixE2EForkEpoch

	// Terminal Total Difficulty.
	e2eConfig.TerminalTotalDifficulty = "616"

	// Prysm constants.
	e2eConfig.ConfigName = EndToEndMainnetName
	e2eConfig.GenesisForkVersion = []byte{0, 0, 0, 254}
	e2eConfig.AltairForkVersion = []byte{1, 0, 0, 254}
	e2eConfig.BellatrixForkVersion = []byte{2, 0, 0, 254}
	e2eConfig.ShardingForkVersion = []byte{3, 0, 0, 254}

	e2eConfig.InitializeForkSchedule()
	return e2eConfig
}

// E2EMainnetConfigYaml returns the e2e config in yaml format.
func E2EMainnetConfigYaml() []byte {
	return ConfigToYaml(E2EMainnetTestConfig())
}

// E2ETestConfigYaml returns the e2e config in yaml format.
func E2ETestConfigYaml() []byte {
	return ConfigToYaml(E2ETestConfig())
}
