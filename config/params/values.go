package params

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
)

const (
	Mainnet ConfigName = iota
	Minimal
	EndToEnd
	Prater
	EndToEndMainnet
	Dynamic
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
	Prater:          "prater",
	EndToEndMainnet: "end-to-end-mainnet",
	Dynamic:         "dynamic",
}
var reverseConfigNames map[string]ConfigName

// KnownConfigs provides an index of all known BeaconChainConfig values.
var KnownConfigs = map[ConfigName]func() *BeaconChainConfig{
	Mainnet:         MainnetConfig,
	Prater:          PraterConfig,
	Minimal:         MinimalSpecConfig,
	EndToEnd:        E2ETestConfig,
	EndToEndMainnet: E2EMainnetTestConfig,
}

var knownForkVersions map[[fieldparams.VersionLength]byte]ConfigName

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
	err := rebuildKnownForkVersions()
	if err != nil {
		panic(err)
	}
	buildReverseConfigName()
}

func buildReverseConfigName() {
	reverseConfigNames = make(map[string]ConfigName)
	for cn, s := range ConfigNames {
		reverseConfigNames[s] = cn
	}
}

var rblock sync.Mutex

func rebuildKnownForkVersions() error {
	rblock.Lock()
	defer rblock.Unlock()
	knownForkVersions = make(map[[fieldparams.VersionLength]byte]ConfigName)
	for n, cfunc := range KnownConfigs {
		cfg := cfunc()
		// ensure that fork schedule is consistent w/ struct fields for all known configurations
		if err := equalForkSchedules(configForkSchedule(cfg), cfg.ForkVersionSchedule); err != nil {
			return errors.Wrapf(err, "improperly initialized fork schedule for config %s", n.String())
		}
		// ensure that all fork versions are unique
		for v := range cfg.ForkVersionSchedule {
			pn, exists := knownForkVersions[v]
			if exists {
				previous := KnownConfigs[pn]()
				return fmt.Errorf("version %#x is duplicated in 2 configs, %s at epoch %d, %s at epoch %d",
					v, pn, previous.ForkVersionSchedule[v], n, cfg.ForkVersionSchedule[v])
			}
			knownForkVersions[v] = n
		}
	}
	return nil
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

var TestForkVersionSuffix byte = 252
var MainnetForkVersionSuffix byte = 0

func SetTestForkVersions(cfg *BeaconChainConfig, suffix byte) {
	cfg.GenesisForkVersion = []byte{0, 0, 0, suffix}
	cfg.AltairForkVersion = []byte{1, 0, 0, suffix}
	cfg.BellatrixForkVersion = []byte{2, 0, 0, suffix}
	cfg.ShardingForkVersion = []byte{3, 0, 0, suffix}
}
