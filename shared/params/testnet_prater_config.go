package params

import (
	eth1Params "github.com/ethereum/go-ethereum/params"
)

// UsePraterNetworkConfig uses the Prater specific
// network config.
func UsePraterNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 4367322
	cfg.BootstrapNodes = []string{
		// TODO(8612): Define bootstrap nodes.
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UsePraterConfig sets the main beacon chain
// config for Prater.
func UsePraterConfig() {
	beaconConfig = PraterConfig()
}

// PraterConfig defines the config for the
// Prater testnet.
func PraterConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1614588812
	cfg.GenesisDelay = 1919188
	cfg.ConfigName = ConfigNames[Prater]
	cfg.GenesisForkVersion = []byte{0x00, 0x00, 0x10, 0x20}
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = eth1Params.GoerliChainConfig.ChainID.Uint64()
	cfg.DepositNetworkID = eth1Params.GoerliChainConfig.ChainID.Uint64()
	cfg.DepositContractAddress = "0xff50ed3d0ec03ac01d4c79aad74928bff48a7b2b"
	return cfg
}
