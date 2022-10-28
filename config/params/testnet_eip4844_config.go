package params

// UseEIP4844NetworkConfig uses the EIP4844 beacon chain specific network config.
func UseEIP4844NetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.MinEpochsForBlobsSidecarsRequest = 1200 // 1 day
	cfg.ContractDeploymentBlock = 0             // deposit contract is a predeploy
	cfg.BootstrapNodes = []string{
		"enr:-MK4QFURnlP5nu_JHdrj6XVYPo4an3tLVD3Ii_hLpFxAvdaVVLOOHPzmAYQQ4lk1U2fwb4oQIh-lYL3UbpTGYr-yJjKGAYO2dGzih2F0dG5ldHOIAAAAAAAAAACEZXRoMpCcZxEogwAP_f__________gmlkgnY0gmlwhCJ5ITWJc2VjcDI1NmsxoQIlwaxycUgJ_Ht4lYdDlInbIuRxu0HcHcFbu0D7As2SLYhzeW5jbmV0cwCDdGNwgjLIg3VkcIIu4A",
		"enr:-MK4QCC-n6C8hHOsUacSgYR7E2UknE_Slz5Tt8h0FiSKxiXDBrki2iwIALq9FIPreXp2GgFJqFM4Bd-1oMlrHgOPKY2GAYO2dG08h2F0dG5ldHOIAAAACAAAAACEZXRoMpCcZxEogwAP_f__________gmlkgnY0gmlwhCJ6vpeJc2VjcDI1NmsxoQNJzjxNKr7-a-iEDs0KvaL_vo1UH91kefEiWzgAdwSntYhzeW5jbmV0cw-DdGNwgjLIg3VkcIIu4A",
		"enr:-MK4QBRIqJE6bT7janDe8o3l_bW20WVtZBqqaxUNvsrHrKYjOUNEG1DFgj-9aOOrRzkZxRgczQZacChxObWGq1X3q3CGAYO2dGzHh2F0dG5ldHOIAAAAAAAAACCEZXRoMpCcZxEogwAP_f__________gmlkgnY0gmlwhCKtCCuJc2VjcDI1NmsxoQJb4OOCku4riQKTyRXbWh0ooc_NXFlP6Y1_A5imyBJcoohzeW5jbmV0cw-DdGNwgjLIg3VkcIIu4A",
		// TODO(EIP-4844): Coinbase boot node
	}
	OverrideBeaconNetworkConfig(cfg)
}

// EIP4844Config defines the config for the EIP4844 beacon chain testnet.
func EIP4844Config() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1653318000
	cfg.MinGenesisActiveValidatorCount = 2
	cfg.Eth1FollowDistance = 15
	cfg.ConfigName = EIP4844Name
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x0f, 0xfd}
	cfg.SecondsPerETH1Block = 12
	cfg.DepositChainID = 1332
	cfg.DepositNetworkID = 70
	cfg.AltairForkEpoch = 1
	cfg.AltairForkVersion = []byte{0x01, 0x00, 0x0f, 0xfd}
	cfg.BellatrixForkEpoch = 2
	cfg.BellatrixForkVersion = []byte{0x02, 0x00, 0x0f, 0xfd}
	cfg.Eip4844ForkEpoch = 3
	cfg.SlotsPerEpoch = 8 // 96 secs; reduced from 32 (6.4 mins) for testing
	cfg.Eip4844ForkVersion = []byte{0x83, 0x00, 0x0f, 0xfd}
	cfg.TerminalTotalDifficulty = "2"
	cfg.DepositContractAddress = "0x8A04d14125D0FDCDc742F4A05C051De07232EDa4"
	cfg.DomainBlobsSidecar = [4]byte{0x0a, 0x00, 0x00, 0x00}
	cfg.InitializeForkSchedule()
	return cfg
}
