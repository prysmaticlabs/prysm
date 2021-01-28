package params

// UseToledoNetworkConfig uses the Toledo specific
// network config.
func UseToledoNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3702432
	cfg.BootstrapNodes = []string{
		// Prysm Bootnode 1
		"enr:-Ku4QL5E378NT4-vqP6v1mZ7kHxiTHJvuBvQixQsuTTCffa0PJNWMBlG3Mduvsvd6T2YP1U3l5tBKO5H-9wyX2SCtPkBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC4EvfsAHAe0P__________gmlkgnY0gmlwhDaetEeJc2VjcDI1NmsxoQKtGC2CAuba7goLLdle899M3esUmoWRvzi7GBVhq6ViCYN1ZHCCIyg",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UseToledoConfig sets the main beacon chain
// config for Toledo testnet.
func UseToledoConfig() {
	beaconConfig = ToledoConfig()
}

// ToledoConfig defines the config for the
// Toledo testnet.
func ToledoConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1605009600
	cfg.GenesisDelay = 86400
	cfg.GenesisForkVersion = []byte{0x00, 0x70, 0x1E, 0xD0}
	cfg.ConfigName = "toledo"
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = 5
	cfg.DepositNetworkID = 5
	cfg.DepositContractAddress = "0x47709dC7a8c18688a1f051761fc34ac253970bC0"
	return cfg
}
