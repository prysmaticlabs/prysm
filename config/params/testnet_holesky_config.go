package params

import "math"

// UseHoleskyNetworkConfig uses the Holesky beacon chain specific network config.
func UseHoleskyNetworkConfig() {
	cfg := BeaconNetworkConfig().Copy()
	cfg.ContractDeploymentBlock = 0
	cfg.BootstrapNodes = []string{
		// EF
		"enr:-Iq4QJk4WqRkjsX5c2CXtOra6HnxN-BMXnWhmhEQO9Bn9iABTJGdjUOurM7Btj1ouKaFkvTRoju5vz2GPmVON2dffQKGAX53x8JigmlkgnY0gmlwhLKAlv6Jc2VjcDI1NmsxoQK6S-Cii_KmfFdUJL2TANL3ksaKUnNXvTCv1tLwXs0QgIN1ZHCCIyk",
		"enr:-KG4QMH842KsJOZAHxI98VJcf8oPr1U8Ylyp2Tb-sNAPniWSCaxIS4F9gc3lGOnROEok7g5qrOm8WgJTl2WXx8MhMmIMhGV0aDKQqX6DZjABcAAKAAAAAAAAAIJpZIJ2NIJpcISygIjpiXNlY3AyNTZrMaECvQMvoDF46BfJgvAbbv1hwpNu9VQBXRIpHS_B8zmkZmmDdGNwgiMog3VkcIIjKA",
		"enr:-Ly4QDU8tZeygxz1gEeAD4EKe4H_8gg-IanpTY6h8A1YGPv5BPNvCMD77zjHUk_iF1pfG_8DC6jYWbIOD1k5kF-LaG4Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpCpfoNmMAFwAAoAAAAAAAAAgmlkgnY0gmlwhJK-DYCJc2VjcDI1NmsxoQN4bUae9DwIcq_56DNztksQYXeddTDKRonI5qI3YhN4SohzeW5jbmV0cwCDdGNwgiMog3VkcIIjKA",
		// Teku
		"enr:-LK4QMlzEff6d-M0A1pSFG5lJ2c56i_I-ZftdojZbW3ehkGNM4pkQuHQqzVvF1BG9aDjIakjnmO23mCBFFZ2w5zOsugEh2F0dG5ldHOIAAAAAAYAAACEZXRoMpCpfoNmMAFwAAABAAAAAAAAgmlkgnY0gmlwhKyuI_mJc2VjcDI1NmsxoQIH1kQRCZW-4AIVyAeXj5o49m_IqNFKRHp6tSpfXMUrSYN0Y3CCIyiDdWRwgiMo",
	}
	OverrideBeaconNetworkConfig(cfg)
}

// HoleskyConfig defines the config for the Holesky beacon chain testnet.
func HoleskyConfig() *BeaconChainConfig {
	cfg := MainnetConfig().Copy()
	cfg.MinGenesisTime = 1694786100
	cfg.GenesisDelay = 300
	cfg.ConfigName = HoleskyName
	cfg.GenesisForkVersion = []byte{0x00, 0x01, 0x70, 0x00}
	cfg.SecondsPerETH1Block = 14
	cfg.DepositChainID = 17000
	cfg.DepositNetworkID = 17000
	cfg.AltairForkEpoch = 0
	cfg.AltairForkVersion = []byte{0x10, 0x1, 0x70, 0x0}
	cfg.BellatrixForkEpoch = 0
	cfg.BellatrixForkVersion = []byte{0x20, 0x1, 0x70, 0x0}
	cfg.CapellaForkEpoch = 256
	cfg.CapellaForkVersion = []byte{0x30, 0x1, 0x70, 0x0}
	cfg.DenebForkEpoch = math.MaxUint64
	cfg.DenebForkVersion = []byte{0x40, 0x1, 0x70, 0x0}
	cfg.TerminalTotalDifficulty = "0"
	cfg.DepositContractAddress = "0x4242424242424242424242424242424242424242"
	cfg.EjectionBalance = 28000000000
	cfg.MaxBlobsPerBlock = 6
	cfg.InitializeForkSchedule()
	return cfg
}
