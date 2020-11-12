package params

// UsePyrmontNetworkConfig uses the Pyrmont specific
// network config.
func UsePyrmontNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3713500
	cfg.ChainID = 5
	cfg.NetworkID = 5
	cfg.DepositContractAddress = "0x2c539a95d2a3f9b10681D9c0dD7cCE37D40c7B79"
	cfg.BootstrapNodes = []string{
		"enr:-Ku4QDuuQGbUpzWMW1IUZpvt3xUzZuEwm2CvHqWQ-EGGzWXPYNc-PZPIfm05R7W3YwEIGM2_2-Y3JHQuEiizbYlW-HoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhDQPSjiJc2VjcDI1NmsxoQM6yTQB6XGWYJbI7NZFBjp4Yb9AYKQPBhVrfUclQUobb4N1ZHCCIyg",
		"enr:-Ku4QAOnRymufUy7UbyxheWFbV9WAtt7BlvoixBz8-Xstb0oBui0ERAiBcsY5xDbE2YxvT7u6gwZPju9V_ecAAJMddUBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhDaa13aJc2VjcDI1NmsxoQKdNQJvnohpf0VO0ZYCAJxGjT0uwJoAHbAiBMujGjK0SoN1ZHCCIyg",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UsePyrmontConfig sets the main beacon chain
// config for Pyrmont.
func UsePyrmontConfig() {
	beaconConfig = PyrmontConfig()
}

// PyrmontConfig defines the config for the
// Pyrmont testnet.
func PyrmontConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1605700800
	cfg.NetworkName = "pyrmont"
	cfg.SecondsPerETH1Block = 14
	return cfg
}
