package params

import (
	"fmt"
	"strings"
)

const altairE2EForkEpoch = 6

// UseE2EConfig for beacon chain services.
func UseE2EConfig() {
	beaconConfig = E2ETestConfig()

	cfg := BeaconNetworkConfig().Copy()
	OverrideBeaconNetworkConfig(cfg)
}

// UseE2EConfig for beacon chain services.
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

	return e2eConfig
}

func E2EMainnetTestConfig() *BeaconChainConfig {
	e2eConfig := MainnetConfig().Copy()

	// Misc.
	e2eConfig.MinGenesisActiveValidatorCount = 256
	e2eConfig.GenesisDelay = 10 // 10 seconds so E2E has enough time to process deposits and get started.
	e2eConfig.ChurnLimitQuotient = 65536

	// Time parameters.
	e2eConfig.SecondsPerSlot = 5
	//e2eConfig.SlotsPerEpoch = 6
	e2eConfig.SqrRootSlotsPerEpoch = 5
	e2eConfig.SecondsPerETH1Block = 2
	e2eConfig.Eth1FollowDistance = 4
	//e2eConfig.EpochsPerEth1VotingPeriod = 2
	e2eConfig.ShardCommitteePeriod = 4
	e2eConfig.MaxSeedLookahead = 1

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
	cfg := E2EMainnetTestConfig()
	lines := []string{}
	lines = append(lines, fmt.Sprintf("PRESET_BASE: '%s'", cfg.PresetBase))
	lines = append(lines, fmt.Sprintf("CONFIG_NAME: '%s'", cfg.ConfigName))
	lines = append(lines, fmt.Sprintf("MIN_GENESIS_ACTIVE_VALIDATOR_COUNT: %d", cfg.MinGenesisActiveValidatorCount))
	lines = append(lines, fmt.Sprintf("GENESIS_DELAY: %d", cfg.GenesisDelay))
	lines = append(lines, fmt.Sprintf("MIN_GENESIS_TIME: %d", cfg.MinGenesisTime))
	lines = append(lines, fmt.Sprintf("GENESIS_FORK_VERSION: %#x", cfg.GenesisForkVersion))
	lines = append(lines, fmt.Sprintf("CHURN_LIMIT_QUOTIENT: %d", cfg.ChurnLimitQuotient))
	lines = append(lines, fmt.Sprintf("SECONDS_PER_SLOT: %d", cfg.SecondsPerSlot))
	lines = append(lines, fmt.Sprintf("SLOTS_PER_EPOCH: %d", cfg.SlotsPerEpoch))
	lines = append(lines, fmt.Sprintf("SECONDS_PER_ETH1_BLOCK: %d", cfg.SecondsPerETH1Block))
	lines = append(lines, fmt.Sprintf("ETH1_FOLLOW_DISTANCE: %d", cfg.Eth1FollowDistance))
	lines = append(lines, fmt.Sprintf("EPOCHS_PER_ETH1_VOTING_PERIOD: %d", cfg.EpochsPerEth1VotingPeriod))
	lines = append(lines, fmt.Sprintf("SHARD_COMMITTEE_PERIOD: %d", cfg.ShardCommitteePeriod))
	lines = append(lines, fmt.Sprintf("MIN_VALIDATOR_WITHDRAWABILITY_DELAY: %d", cfg.MinValidatorWithdrawabilityDelay))
	lines = append(lines, fmt.Sprintf("MAX_SEED_LOOKAHEAD: %d", cfg.MaxSeedLookahead))
	lines = append(lines, fmt.Sprintf("EJECTION_BALANCE: %d", cfg.EjectionBalance))
	lines = append(lines, fmt.Sprintf("MIN_PER_EPOCH_CHURN_LIMIT: %d", cfg.MinPerEpochChurnLimit))
	lines = append(lines, fmt.Sprintf("DEPOSIT_CHAIN_ID: %d", cfg.DepositChainID))
	lines = append(lines, fmt.Sprintf("DEPOSIT_NETWORK_ID: %d", cfg.DepositNetworkID))
	lines = append(lines, fmt.Sprintf("ALTAIR_FORK_EPOCH: %d", cfg.AltairForkEpoch))
	lines = append(lines, fmt.Sprintf("ALTAIR_FORK_VERSION: %#x", cfg.AltairForkVersion))
	lines = append(lines, fmt.Sprintf("INACTIVITY_SCORE_BIAS: %d", cfg.InactivityScoreBias))
	lines = append(lines, fmt.Sprintf("INACTIVITY_SCORE_RECOVERY_RATE: %d", cfg.InactivityScoreRecoveryRate))

	yamlFile := []byte(strings.Join(lines, "\n"))
	return yamlFile
}
