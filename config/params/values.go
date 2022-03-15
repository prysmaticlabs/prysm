package params

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Prater
)

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	Minimal:  "minimal",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Prater:   "prater",
}

// ConfigName enum describes the type of known network in use.
type ConfigName int

func (n ConfigName) String() string {
	s, ok := ConfigNames[n]
	if !ok {
		return "undefined"
	}
	return s
}

func AllConfigs() map[ConfigName]*BeaconChainConfig {
	all := make(map[ConfigName]*BeaconChainConfig)
	for name := range ConfigNames {
		var cfg *BeaconChainConfig
		switch name {
		case Mainnet:
			cfg = MainnetConfig()
		case Prater:
			cfg = PraterConfig()
		case Pyrmont:
			cfg = PyrmontConfig()
		case Minimal:
			cfg = MinimalSpecConfig()
		case EndToEnd:
			cfg = E2ETestConfig()
		}
		cfg = cfg.Copy()
		cfg.InitializeForkSchedule()
		all[name] = cfg
	}
	return all
}

type ForkName int

const (
	ForkGenesis ForkName = iota
	ForkAltair
	ForkBellatrix
)

func (n ForkName) String() string {
	switch n {
	case ForkGenesis:
		return "genesis"
	case ForkAltair:
		return "altair"
	case ForkBellatrix:
		return "bellatrix"
	}

	return "undefined"
}
