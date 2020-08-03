package params

// UseMedallaNetworkConfig uses the Medalla specific
// network config.
func UseMedallaNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 3085928
	cfg.DepositContractAddress = "0x07b39F4fDE4A38bACe212b546dAc87C58DfE3fDC"
	cfg.ChainID = 5   // Chain ID of eth1 goerli testnet.
	cfg.NetworkID = 5 // Network ID of eth1 goerli testnet.
	// Medalla Bootstrap Nodes
	cfg.BootstrapNodes = []string{
		// Prylabs Bootnodes
		"enr:-Ku4QLglCMIYAgHd51uFUqejD9DWGovHOseHQy7Od1SeZnHnQ3fSpE4_nbfVs8lsy8uF07ae7IgrOOUFU0NFvZp5D4wBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpAYrkzLAAAAAf__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A",
		"enr:-Ku4QOdk3u7rXI5YvqwmEbApW_OLlRkq_yzmmhdlrJMcfviacLWwSm-tr1BOvamuRQqfc6lnMeec4E4ddOhd3KqCB98Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpAYrkzLAAAAAf__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQKH3lxnglLqrA7L6sl5r7XFnckr3XCnlZMaBTYSdE8SHIN1ZHCCG1g",
		"enr:-Ku4QOVrqhlmsh9m2MGSnvVz8XPfjwHWBuOcgVQvWwBhN0-NI0XVhSerujBBwIeLpc-OES0C9iAzJhiCgRZ0xH13DgEBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpAYrkzLAAAAAf__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQLEq16KLm1vPjUKYGkHq296D60i7y209NYPUpwZPXDVgYN1ZHCCF3A",
		// External Bootnodes
		"enr:-Ku4QFVactU18ogiqPPasKs3jhUm5ISszUrUMK2c6SUPbGtANXVJ2wFapsKwVEVnVKxZ7Gsr9yEc4PYF-a14ahPa1q0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpAYrkzLAAAAAf__________gmlkgnY0gmlwhGQbAHyJc2VjcDI1NmsxoQILF-Ya2i5yowVkQtlnZLjG0kqC4qtwmSk8ha7tKLuME4N1ZHCCIyg",
		"enr:-KG4QFuKQ9eeXDTf8J4tBxFvs3QeMrr72mvS7qJgL9ieO6k9Rq5QuGqtGK4VlXMNHfe34Khhw427r7peSoIbGcN91fUDhGV0aDKQD8XYjwAAAAH__________4JpZIJ2NIJpcIQDhMExiXNlY3AyNTZrMaEDESplmV9c2k73v0DjxVXJ6__2bWyP-tK28_80lf7dUhqDdGNwgiMog3VkcIIjKA",
	}

	OverrideBeaconNetworkConfig(cfg)
}

// MedallaConfig defines the config for the
// medalla testnet.
func MedallaConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1596546000
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x00, 0x01}
	return cfg
}

// UseMedallaConfig sets the main beacon chain
// config for medalla.
func UseMedallaConfig() {
	beaconConfig = MedallaConfig()
}
