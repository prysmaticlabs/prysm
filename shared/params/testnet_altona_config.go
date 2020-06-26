package params

// UseAltonaNetworkConfig uses the Altona specific
// network config.
func UseAltonaNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 2917810
	cfg.DepositContractAddress = "0x16e82D77882A663454Ef92806b7DeCa1D394810f"
	cfg.BootstrapNodes = []string{"enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0idV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"}
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
	altCfg.MinGenesisTime = 1593086400
	altCfg.GenesisForkVersion = []byte{0x00, 0x00, 0x01, 0x21}
	return altCfg
}
