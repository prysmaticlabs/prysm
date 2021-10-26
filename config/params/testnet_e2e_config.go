package params

import (
	"math"

	types "github.com/prysmaticlabs/eth2-types"
)

const (
	altairE2EForkEpoch    = 6
	bellatrixE2EForkEpoch = 8 //nolint:deadcode
)

// UseE2EConfig for beacon chain services.
func UseE2EConfig() {
	beaconConfig = E2ETestConfig()

	cfg := BeaconNetworkConfig().Copy()
	OverrideBeaconNetworkConfig(cfg)
}

// UseE2EMainnetConfig for beacon chain services.
func UseE2EMainnetConfig() {
	beaconConfig = E2EMainnetTestConfig()

	cfg := BeaconNetworkConfig().Copy()
	OverrideBeaconNetworkConfig(cfg)
}

// E2ETestConfig retrieves the configurations made specifically for E2E testing.
// Warning: This config is only for testing, it is not meant for use outside of E2E.
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
	e2eConfig.Eth1FollowDistance = 4
	e2eConfig.EpochsPerEth1VotingPeriod = 2
	e2eConfig.ShardCommitteePeriod = 4
	e2eConfig.MaxSeedLookahead = 1

	// PoW parameters.
	e2eConfig.DepositChainID = 1337   // Chain ID of eth1 dev net.
	e2eConfig.DepositNetworkID = 1337 // Network ID of eth1 dev net.

	// Altair Fork Parameters.
	e2eConfig.AltairForkEpoch = altairE2EForkEpoch

	// Prysm constants.
	e2eConfig.ConfigName = ConfigNames[EndToEnd]

	e2eConfig.GenesisForkVersion = []byte{0x00, 0x00, 0xFF, 0xFF}
	e2eConfig.AltairForkVersion = []byte{0x1, 0x0, 0xFF, 0xFF}
	e2eConfig.ShardingForkVersion = []byte{0x3, 0x0, 0xFF, 0xFF}
	e2eConfig.BellatrixForkVersion = []byte{0x2, 0x0, 0xFF, 0xFF}

	e2eConfig.ForkVersionSchedule = map[[4]byte]types.Epoch{
		{0x00, 0x00, 0xFF, 0xFF}: 0,
		{0x1, 0x0, 0xFF, 0xFF}:   math.MaxUint64,
		{0x3, 0x0, 0xFF, 0xFF}:   math.MaxUint64,
		{0x2, 0x0, 0xFF, 0xFF}:   math.MaxUint64,
	}

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
	e2eConfig.Eth1FollowDistance = 4
	e2eConfig.ShardCommitteePeriod = 4

	// PoW parameters.
	e2eConfig.DepositChainID = 1337   // Chain ID of eth1 dev net.
	e2eConfig.DepositNetworkID = 1337 // Network ID of eth1 dev net.

	// Altair Fork Parameters.
	e2eConfig.AltairForkEpoch = altairE2EForkEpoch

	// Prysm constants.
	e2eConfig.ConfigName = ConfigNames[EndToEnd]

	return e2eConfig
}

// E2EMainnetConfigYaml returns the e2e config in yaml format.
func E2EMainnetConfigYaml() []byte {
	return ConfigToYaml(E2EMainnetTestConfig())
}
