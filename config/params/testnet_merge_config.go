package params

import "math"

// UseMergeTestNetworkConfig uses the Merge specific
// network config.
func UseMergeTestNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0
	cfg.BootstrapNodes = []string{
		"enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UseMergeTestConfig sets the main beacon chain
// config for Merge testnet.
func UseMergeTestConfig() {
	beaconConfig = MergeTestnetConfig()
}

// MergeTestnetConfig defines the config for the
// Merge testnet.
func MergeTestnetConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisActiveValidatorCount = 25000
	cfg.MinGenesisTime = 1638288000
	cfg.GenesisDelay = 300
	cfg.ConfigName = "Merge"
	cfg.GenesisForkVersion = []byte{0x40, 0x00, 0x00, 0x69}
	cfg.AltairForkVersion = []byte{0x41, 0x00, 0x00, 0x70}
	cfg.AltairForkEpoch = 4
	cfg.MergeForkVersion = []byte{0x42, 0x00, 0x00, 0x71}
	cfg.MergeForkEpoch = 10
	cfg.TerminalTotalDifficulty = 100000000
	cfg.ShardingForkVersion = []byte{0x03, 0x00, 0x00, 0x00}
	cfg.ShardingForkEpoch = math.MaxUint64
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = 1337502
	cfg.DepositNetworkID = 1337502
	cfg.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	cfg.Eth1FollowDistance = 16
	return cfg
}
