package params

// UseSpadinaNetworkConfig uses the Spadina specific
// network config.
func UseSpadinaNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3384340
	cfg.ChainID = 5
	cfg.NetworkID = 5
	cfg.DepositContractAddress = "0x48B597F4b53C21B48AD95c7256B49D1779Bd5890"
	cfg.BootstrapNodes = []string{}

	OverrideBeaconNetworkConfig(cfg)
}

// UseSpadinaConfig sets the main beacon chain
// config for Spadina.
func UseSpadinaConfig() {
	beaconConfig = SpadinaConfig()
}

// SpadinaConfig defines the config for the
// Spadina testnet.
func SpadinaConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1601380800
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x00, 0x02}
	cfg.NetworkName = "Spadina"
	cfg.MinGenesisActiveValidatorCount = 1024
	return cfg
}
