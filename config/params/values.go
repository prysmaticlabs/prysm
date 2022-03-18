package params

import (
	"fmt"
	"github.com/pkg/errors"
)

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Prater
)

// ConfigName enum describes the type of known network in use.
type ConfigName int

func (n ConfigName) String() string {
	s, ok := ConfigNames[n]
	if !ok {
		return "undefined"
	}
	return s
}

// ConfigNames provides network configuration names.
var ConfigNames = map[ConfigName]string{
	Mainnet:  "mainnet",
	Minimal:  "minimal",
	EndToEnd: "end-to-end",
	Pyrmont:  "pyrmont",
	Prater:   "prater",
}

// KnownConfigs provides an index of all known BeaconChainConfig values.
var KnownConfigs = map[ConfigName]func() *BeaconChainConfig{
	Mainnet:  MainnetConfig,
	Prater:   PraterConfig,
	Pyrmont:  PyrmontConfig,
	Minimal:  MinimalSpecConfig,
	EndToEnd: E2ETestConfig,
}

var knownForkVersions map[[4]byte]ConfigName

var errUnknownForkVersion = errors.New("version not found in fork version schedule for any known config")

// ConfigForVersion find the BeaconChainConfig corresponding to the version bytes.
// Version bytes for BeaconChainConfig values in KnownConfigs are proven to be unique during package initialization.
func ConfigForVersion(version [4]byte) (*BeaconChainConfig, error) {
	cfg, ok := knownForkVersions[version]
	if !ok {
		return nil, errors.Wrapf(errUnknownForkVersion, "version=%#x", version)
	}
	return KnownConfigs[cfg](), nil
}

func init() {
	knownForkVersions = make(map[[4]byte]ConfigName)
	for n, cfunc := range KnownConfigs {
		cfg := cfunc()
		// ensure that fork schedule is consistent w/ struct fields for all known configurations
		cfg.InitializeForkSchedule()
		// ensure that all fork versions are unique
		for v, _ := range cfg.ForkVersionSchedule {
			pn, exists := knownForkVersions[v]
			if exists {
				previous := KnownConfigs[pn]()
				msg := fmt.Sprintf("version %#x is duplicated in 2 configs, %s at epoch %d, %s at epoch %d",
					v, pn, previous.ForkVersionSchedule[v], n, cfg.ForkVersionSchedule[v])
				panic(msg)
			}
			knownForkVersions[v] = n
		}
	}
}
