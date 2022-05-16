package params

import (
	eth1Params "github.com/ethereum/go-ethereum/params"
)

// UseBopstenNetworkConfig uses the Ropsten beacon chain(Bopsten) specific network config.
func UseBopstenNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 12269949
	cfg.BootstrapNodes = []string{
		"enr:-Iq4QMCTfIMXnow27baRUb35Q8iiFHSIDBJh6hQM5Axohhf4b6Kr_cOCu0htQ5WvVqKvFgY28893DHAg8gnBAXsAVqmGAX53x8JggmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UseBopstenConfig sets the main beacon chain config for Ropsten beacon chain.
func UseBopstenConfig() {
	beaconConfig = BopstenConfig()
}

// BopstenConfig defines the config for the Robsten beacon chain testnet.
func BopstenConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1653922800
	cfg.GenesisDelay = 300
	cfg.ConfigName = BopstenName
	cfg.GenesisForkVersion = []byte{0x80, 0x00, 0x00, 0x69}
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = eth1Params.RopstenChainConfig.ChainID.Uint64()
	cfg.DepositNetworkID = eth1Params.RopstenChainConfig.ChainID.Uint64()
	cfg.AltairForkEpoch = 500
	cfg.AltairForkVersion = []byte{0x80, 0x00, 0x00, 0x70}
	cfg.BellatrixForkEpoch = 750
	cfg.BellatrixForkVersion = []byte{0x80, 0x00, 0x00, 0x71}
	cfg.TerminalTotalDifficulty = "43531756765713534"
	cfg.DepositContractAddress = "0x6f22fFbC56eFF051aECF839396DD1eD9aD6BBA9D"
	cfg.InitializeForkSchedule()
	return cfg
}
