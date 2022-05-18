package params

// InteropConfig provides a generic config suitable for interop testing.
func InteropConfig() *BeaconChainConfig {
	e2eConfig := MainnetConfig().Copy()

	// Prysm constants.
	e2eConfig.ConfigName = InteropName
	e2eConfig.GenesisForkVersion = []byte{0, 0, 0, 235}
	e2eConfig.AltairForkVersion = []byte{1, 0, 0, 235}
	e2eConfig.BellatrixForkVersion = []byte{2, 0, 0, 235}
	e2eConfig.ShardingForkVersion = []byte{3, 0, 0, 235}

	e2eConfig.InitializeForkSchedule()
	return e2eConfig
}
