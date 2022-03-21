package params

import (
	"fmt"
	types "github.com/prysmaticlabs/eth2-types"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"

	"github.com/pkg/errors"
)

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Pyrmont
	Prater
	EndToEndMainnet
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
	Mainnet:         "mainnet",
	Minimal:         "minimal",
	EndToEnd:        "end-to-end",
	Pyrmont:         "pyrmont",
	Prater:          "prater",
	EndToEndMainnet: "end-to-end-mainnet",
}

// KnownConfigs provides an index of all known BeaconChainConfig values.
var KnownConfigs = map[ConfigName]func() *BeaconChainConfig{
	Mainnet:         MainnetConfig,
	Prater:          PraterConfig,
	Pyrmont:         PyrmontConfig,
	Minimal:         MinimalSpecConfig,
	EndToEnd:        E2ETestConfig,
	EndToEndMainnet: E2EMainnetTestConfig,
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
		if err := equalForkSchedules(configForkSchedule(cfg), cfg.ForkVersionSchedule); err != nil {
			panic(errors.Wrapf(err, "improperly initialized for schedule for config %s", n.String()))
		}
		// ensure that all fork versions are unique
		for v := range cfg.ForkVersionSchedule {
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

func equalForkSchedules(a, b map[[fieldparams.VersionLength]byte]types.Epoch) error {
	if len(a) != len(b) {
		return fmt.Errorf("different lengths, a=%d, b=%d", len(a), len(b))
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return fmt.Errorf("fork version %#x in a not present in b")
		}
		if v != bv {
			return fmt.Errorf("fork version mismatch, epoch in a=%d, b=%d", v, bv)
		}
	}
	return nil
}
