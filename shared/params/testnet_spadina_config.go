package params

// UseSpadinaNetworkConfig uses the Spadina specific
// network config.
func UseSpadinaNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3384340
	cfg.ChainID = 5
	cfg.NetworkID = 5
	cfg.DepositContractAddress = "0x48B597F4b53C21B48AD95c7256B49D1779Bd5890"
	cfg.BootstrapNodes = []string{
		// Prysm Bootnode 1
		"enr:-Ku4QGQJf2bcDAwVGvbvtq3AB4KKwAvStTenY-i_QnW2ABNRRBncIU_5qR_e_um-9t3s9g-Y5ZfFATj1nhtzq6lvgc4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpDEqCQNAAAAAv__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQNoed9JnQh7ltcAacHEGOjwocL1BhMQbYTgaPX0kFuXtIN1ZHCCE4g",
		// Teku Bootnode 1
		"enr:-KG4QA-EcFfXQsL2dcneG8vp8HTWLrpwHQ5HhfyIytfpeKOISzROy2kYSsf_v-BZKnIx5XHDjqJ-ttz0hoz6qJA7tasEhGV0aDKQxKgkDQAAAAL__________4JpZIJ2NIJpcIQDFt-UiXNlY3AyNTZrMaECkR4C5DVO_9rB48eHTY4kdyOHsguTEDlvb7Ce0_mvghSDdGNwgiMog3VkcIIjKA",
	}
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
