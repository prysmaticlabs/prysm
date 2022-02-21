package params

// UseMergeTestNetworkConfig uses the Merge specific
// network config.
func UseMergeTestNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0
	cfg.BootstrapNodes = []string{}
	OverrideBeaconNetworkConfig(cfg)
}

// UseMergeTestConfig sets the main beacon chain
// config for Merge testnet.
func UseMergeTestConfig() {
	beaconConfig = KilnTestnetConfig()
}

// KilnTestnetConfig defines the config for the Kiln testnet.
func KilnTestnetConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.AltairForkEpoch = 1
	cfg.BellatrixForkEpoch = 2
	cfg.MiniDankShardingForkEpoch = 3
	cfg.DepositChainID = 1337702
	cfg.SecondsPerSlot = 6
	cfg.SlotsPerEpoch = 6
	cfg.TerminalTotalDifficulty = "9"
	cfg.AltairForkVersion = []byte{0x1, 0x0, 0x10, 0x20}
	cfg.BellatrixForkVersion = []byte{0x2, 0x0, 0x10, 0x20}
	cfg.MiniDankShardingForkVersion = []byte{0x3, 0x0, 0x10, 0x20}
	return cfg
}
