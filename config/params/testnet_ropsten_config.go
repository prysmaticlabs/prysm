package params

import (
	eth1Params "github.com/ethereum/go-ethereum/params"
)

// UseRopstenNetworkConfig uses the Ropsten beacon chain specific network config.
func UseRopstenNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 12269949
	cfg.BootstrapNodes = []string{
		// EF boot node
		"enr:-Iq4QMCTfIMXnow27baRUb35Q8iiFHSIDBJh6hQM5Axohhf4b6Kr_cOCu0htQ5WvVqKvFgY28893DHAg8gnBAXsAVqmGAX53x8JggmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk",
		// Teku boot node
		"enr:-KG4QMJSJ7DHk6v2p-W8zQ3Xv7FfssZ_1E3p2eY6kN13staMObUonAurqyWhODoeY6edXtV8e9eL9RnhgZ9va2SMDRQMhGV0aDKQS-iVMYAAAHD0AQAAAAAAAIJpZIJ2NIJpcIQDhAAhiXNlY3AyNTZrMaEDXBVUZhhmdy1MYor1eGdRJ4vHYghFKDgjyHgt6sJ-IlCDdGNwgiMog3VkcIIjKA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// RopstenConfig defines the config for the Ropsten beacon chain testnet.
func RopstenConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1653318000
	cfg.GenesisDelay = 604800
	cfg.MinGenesisActiveValidatorCount = 100000
	cfg.ConfigName = RopstenName
	cfg.GenesisForkVersion = []byte{0x80, 0x00, 0x00, 0x69}
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = eth1Params.RopstenChainConfig.ChainID.Uint64()
	cfg.DepositNetworkID = eth1Params.RopstenChainConfig.ChainID.Uint64()
	cfg.AltairForkEpoch = 500
	cfg.AltairForkVersion = []byte{0x80, 0x00, 0x00, 0x70}
	cfg.BellatrixForkEpoch = 750
	cfg.BellatrixForkVersion = []byte{0x80, 0x00, 0x00, 0x71}
	cfg.CapellaForkVersion = []byte{0x80, 0x00, 0x00, 0x72}
	cfg.TerminalTotalDifficulty = "50000000000000000"
	cfg.DepositContractAddress = "0x6f22fFbC56eFF051aECF839396DD1eD9aD6BBA9D"
	cfg.InitializeForkSchedule()
	return cfg
}
