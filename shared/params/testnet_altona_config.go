package params

// UseAltonaNetworkConfig uses the Altona specific
// network config.
func UseAltonaNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 2917810
	cfg.DepositContractAddress = "0x16e82D77882A663454Ef92806b7DeCa1D394810f"
	cfg.BootstrapNodes = []string{
		"enr:-LK4QFtV7Pz4reD5a7cpfi1z6yPrZ2I9eMMU5mGQpFXLnLoKZW8TXvVubShzLLpsEj6aayvVO1vFx-MApijD3HLPhlECh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD6etXjAAABIf__________gmlkgnY0gmlwhDMPYfCJc2VjcDI1NmsxoQIerw_qBc9apYfZqo2awiwS930_vvmGnW2psuHsTzrJ8YN0Y3CCIyiDdWRwgiMo",
		"enr:-LK4QPVkFd_MKzdW0219doTZryq40tTe8rwWYO75KDmeZM78fBskGsfCuAww9t8y3u0Q0FlhXOhjE1CWpx3SGbUaU80Ch2F0dG5ldHOIAAAAAAAAAACEZXRoMpD6etXjAAABIf__________gmlkgnY0gmlwhDMPRgeJc2VjcDI1NmsxoQNHu-QfNgzl8VxbMiPgv6wgAljojnqAOrN18tzJMuN8oYN0Y3CCIyiDdWRwgiMo",
	}
	
	OverrideBeaconNetworkConfig(cfg)
}

// UseAltonaConfig sets the main beacon chain
// config for altona.
func UseAltonaConfig() {
	beaconConfig = AltonaConfig()
}

// AltonaConfig defines the config for the
// altona testnet.
func AltonaConfig() *BeaconChainConfig {
	altCfg := MainnetConfig().Copy()
	altCfg.MinGenesisActiveValidatorCount = 640
	altCfg.MinGenesisTime = 1593433800
	altCfg.GenesisForkVersion = []byte{0x00, 0x00, 0x01, 0x21}
	return altCfg
}
