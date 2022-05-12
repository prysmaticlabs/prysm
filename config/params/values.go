package params

const (
	EndToEndName        = "end-to-end"
	EndToEndMainnetName = "end-to-end-mainnet"
	MainnetName         = "mainnet"
	MinimalName         = "minimal"
	PraterName          = "prater"
	DevnetName          = "devnet"
	MainnetTestName     = "mainnet-test"
)

// KnownConfigs provides an index of all known BeaconChainConfig values.
var KnownConfigs = map[string]func() *BeaconChainConfig{
	MainnetName:         MainnetConfig,
	PraterName:          PraterConfig,
	MinimalName:         MinimalSpecConfig,
	EndToEndName:        E2ETestConfig,
	EndToEndMainnetName: E2EMainnetTestConfig,
}
