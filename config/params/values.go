package params

import (
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
)

const (
	EndToEndName        = "end-to-end"
	EndToEndMainnetName = "end-to-end-mainnet"
	MainnetName         = "mainnet"
	MinimalName         = "minimal"
	PraterName          = "prater"
)

// KnownConfigs provides an index of all known BeaconChainConfig values.
var KnownConfigs = map[string]func() *BeaconChainConfig{
	MainnetName:         MainnetConfig,
	PraterName:          PraterConfig,
	MinimalName:         MinimalSpecConfig,
	EndToEndName:        E2ETestConfig,
	EndToEndMainnetName: E2EMainnetTestConfig,
}

var knownForkVersions map[[fieldparams.VersionLength]byte]string

var errUnknownForkVersion = errors.New("version not found in fork version schedule for any known config")

// ConfigForVersion find the BeaconChainConfig corresponding to the version bytes.
// Version bytes for BeaconChainConfig values in KnownConfigs are proven to be unique during package initialization.
func ConfigForVersion(version [fieldparams.VersionLength]byte) (*BeaconChainConfig, error) {
	cfg, ok := knownForkVersions[version]
	if !ok {
		return nil, errors.Wrapf(errUnknownForkVersion, "version=%#x", version)
	}
	return KnownConfigs[cfg](), nil
}

func init() {
	knownForkVersions = make(map[[fieldparams.VersionLength]byte]string)
	for n, cfunc := range KnownConfigs {
		cfg := cfunc()
		// ensure that fork schedule is consistent w/ struct fields for all known configurations
		if err := equalForkSchedules(configForkSchedule(cfg), cfg.ForkVersionSchedule); err != nil {
			panic(errors.Wrapf(err, "improperly initialized for schedule for config %s", n))
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
			return fmt.Errorf("fork version %#x from 'a', not present in 'b'", k)
		}
		if v != bv {
			return fmt.Errorf("fork version mismatch, epoch in a=%d, b=%d", v, bv)
		}
	}
	return nil
}
