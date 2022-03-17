package params

import "fmt"

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Prater
)

// AllConfigs is a map of all BeaconChainConfig values, allowing a BeaconChainConfig to be looked up by its ConfigName
var AllConfigs map[ConfigName]*BeaconChainConfig

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

func init() {
	AllConfigs = make(map[ConfigName]*BeaconChainConfig)
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
		default:
			msg := fmt.Sprintf("unknown config '%s' added to ConfigNames, "+
				"please update init() to keep AllConfigs in sync", name)
			panic(msg)
		}
		cfg = cfg.Copy()
		cfg.InitializeForkSchedule()
		AllConfigs[name] = cfg
	}
}
