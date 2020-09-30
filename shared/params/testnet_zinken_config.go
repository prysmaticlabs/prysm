package params

// UseZinkenNetworkConfig uses the Zinken specific
// network config.
func UseZinkenNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3384340
	cfg.ChainID = 5
	cfg.NetworkID = 5
	cfg.DepositContractAddress = "0x99F0Ec06548b086E46Cb0019C78D0b9b9F36cD53"
	// TODO(7390): Add Zinken bootnodes.
	cfg.BootstrapNodes = []string{}
	OverrideBeaconNetworkConfig(cfg)
}

// UseZinkenConfig sets the main beacon chain
// config for Zinken.
func UseZinkenConfig() {
	beaconConfig = ZinkenConfig()
}

// ZinkenConfig defines the config for the
// Zinken testnet.
func ZinkenConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1602504000
	cfg.GenesisDelay = 345600
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x00, 0x03}
	cfg.NetworkName = "zinken"
	cfg.MinGenesisActiveValidatorCount = 1024
	return cfg
}
