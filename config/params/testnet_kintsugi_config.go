package params

import (
	"math"

	"github.com/ethereum/go-ethereum/common"
)

// UseMergeTestNetworkConfig uses the Merge specific
// network config.
func UseMergeTestNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0
	cfg.BootstrapNodes = []string{
		"enr:-Iq4QKuNB_wHmWon7hv5HntHiSsyE1a6cUTK1aT7xDSU_hNTLW3R4mowUboCsqYoh1kN9v3ZoSu_WuvW9Aw0tQ0Dxv6GAXxQ7Nv5gmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk",
		"enr:-KG4QIkKUzDxrv7Xz8u9K9QqoTqEwKKCkLoChxVnfeILU6IdBoWoNOxPGvdl474l1iPFoR8CJUhgGEeO-k1SJ7SJCOEDhGV0aDKQR9ByjGEAAHAKAAAAAAAAAIJpZIJ2NIJpcISl6LnPiXNlY3AyNTZrMaEDprwHy6RKAKJguvGCldiGAI5JDJmQ8TZVnnWQur8zEh2DdGNwgiMog3VkcIIjKA",
		"enr:-Ly4QGJodG8Q0vX5ePXsLsXody1Fbeauyottk3-iAvJ_6XfTVlWGsnfBQPlIOBgexXJqD78bUD5OCnXF5igBBJ4WuboBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpBH0HKMYQAAcAoAAAAAAAAAgmlkgnY0gmlwhEDhBN-Jc2VjcDI1NmsxoQPf98kXQf3Nh3ooc8vBdbUY2WAHR1VDrDhXYTKvRt4n-IhzeW5jbmV0cwCDdGNwgiMog3VkcIIjKA",
		"enr:-KG4QLIhAeEVABV4Id22qEbjemJ0b9JBjRhdYpKN0kvpVi_GbFkQTvAf7-Da-5sW2oNenTW3is_GxLImUCtYzxPMOR4DhGV0aDKQR9ByjGEAAHAKAAAAAAAAAIJpZIJ2NIJpcISl6LF5iXNlY3AyNTZrMaED6XFvht9SUPD0FlYWnjunXhF9FdQMQO56816C9iFNt-WDdGNwgiMog3VkcIIjKA",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// UseMergeTestConfig sets the main beacon chain
// config for Merge testnet.
func UseMergeTestConfig() {
	beaconConfig = KintsugiTestnetConfig()
}

// KintsugiTestnetConfig defines the config for the Kingtsugi testnet.
func KintsugiTestnetConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisActiveValidatorCount = 15000
	cfg.MinGenesisTime = 1639659600
	cfg.GenesisDelay = 300
	cfg.ConfigName = "Merge"
	cfg.GenesisForkVersion = []byte{0x60, 0x00, 0x00, 0x69}
	cfg.AltairForkVersion = []byte{0x61, 0x00, 0x00, 0x70}
	cfg.AltairForkEpoch = 10
	cfg.BellatrixForkVersion = []byte{0x62, 0x00, 0x00, 0x71}
	cfg.BellatrixForkEpoch = 20
	cfg.TerminalTotalDifficulty = "5000000000"
	cfg.TerminalBlockHash = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	cfg.TerminalBlockHashActivationEpoch = 18446744073709551615
	cfg.ShardingForkVersion = []byte{0x03, 0x00, 0x00, 0x00}
	cfg.ShardingForkEpoch = math.MaxUint64
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = 1337702
	cfg.DepositNetworkID = 1337702
	cfg.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	cfg.Eth1FollowDistance = 16
	return cfg
}
