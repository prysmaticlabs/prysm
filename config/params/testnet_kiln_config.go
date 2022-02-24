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
	return cfg
}
