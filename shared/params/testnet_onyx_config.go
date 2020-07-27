package params

// UseOnyxNetworkConfig uses the Onyx specific network config.
func UseOnyxNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 2844925
	cfg.DepositContractAddress = "0x0F0F0fc0530007361933EaB5DB97d09aCDD6C1c8"
	cfg.ChainID = 5   // Chain ID of eth1 goerli testnet.
	cfg.NetworkID = 5 // Network ID of eth1 goerli testnet.
	cfg.BootstrapNodes = []string{"enr:-Ku4QMKVC_MowDsmEa20d5uGjrChI0h8_KsKXDmgVQbIbngZV0idV6_RL7fEtZGo-kTNZ5o7_EJI_vCPJ6scrhwX0Z4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQJxCnE6v_x2ekgY_uoE1rtwzvGy40mq9eD66XfHPBWgIIN1ZHCCD6A"}
	OverrideBeaconNetworkConfig(cfg)
}

// OnyxConfig returns the configuration to be used in the main network. Currently, Onyx uses the
// unchanged mainnet configuration.
func OnyxConfig() *BeaconChainConfig {
	return mainnetBeaconConfig
}

// UseOnyxConfig for beacon chain services. Currently, Onyx uses the unchanged mainnet
// configuration.
func UseOnyxConfig() {
	beaconConfig = MainnetConfig().Copy()
}
