package params

// UseEIP4844NetworkConfig uses the EIP4844 beacon chain specific network config.
func UseEIP4844NetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0 // deposit contract is a predeploy
	cfg.BootstrapNodes = []string{
		// TODO(EIP-4844): Optimism boot node
		// TODO(EIP-4844): Coinbase boot node
	}
	OverrideBeaconNetworkConfig(cfg)
}

// EIP4844Config defines the config for the EIP4844 beacon chain testnet.
func EIP4844Config() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1653318000
	cfg.GenesisDelay = 604800
	cfg.MinGenesisActiveValidatorCount = 1
	cfg.ConfigName = EIP4844Name
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x0f, 0xfd}
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = 1331
	cfg.DepositNetworkID = 69
	cfg.AltairForkEpoch = 1
	cfg.AltairForkVersion = []byte{0x01, 0x00, 0x0f, 0xfd}
	cfg.BellatrixForkEpoch = 2
	cfg.BellatrixForkVersion = []byte{0x02, 0x00, 0x0f, 0xfd}
	cfg.Eip4844ForkEpoch = 3
	cfg.Eip4844ForkVersion = []byte{0x83, 0x00, 0x0f, 0xfd}
	cfg.TerminalTotalDifficulty = "10"
	cfg.DepositContractAddress = "0x8A04d14125D0FDCDc742F4A05C051De07232EDa4"
	cfg.DomainBlobsSidecar = [4]byte{0x0a, 0x00, 0x00, 0x00}
	cfg.InitializeForkSchedule()
	return cfg
}
