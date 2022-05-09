package params

import (
	"sync"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
)

var ErrRegistryCollision = errors.New("registry cannot add config with conflicting fork version schedule")
var ErrConfigNotFound = errors.New("unable to find requested BeaconChainConfig")
var ErrCannotNullifyActive = errors.New("cannot set a config marked as active to nil")
var ErrReplaceNilConfig = errors.New("Replace called with a nil value")
var ErrConfigNameConflict = errors.New("config with conflicting name already exists")

var Registry *registry

type registry struct {
	sync.RWMutex
	active *BeaconChainConfig
	versionToName map[[fieldparams.VersionLength]byte]string
	nameToConfig map[string]*BeaconChainConfig
}

func NewRegistry(configs ...*BeaconChainConfig) *registry {
	r := &registry{
		versionToName: make(map[[fieldparams.VersionLength]byte]string),
		nameToConfig: make(map[string]*BeaconChainConfig),
	}
	for _, c := range configs {
		if err := r.Add(c); err != nil {
			panic(err)
		}
	}
	// ensure that main net is always present and active by default
	if err := r.SetActive(MainnetConfig()); err != nil {
		panic(err)
	}
	return r
}

func (r *registry) Add(c *BeaconChainConfig) error {
	name := c.ConfigName
	if _, exists := r.nameToConfig[name]; exists {
		return errors.Wrapf(ErrConfigNameConflict, "ConfigName=%s", name)
	}
	c.InitializeForkSchedule()
	for v, _ := range c.ForkVersionSchedule {
		if n, exists := r.versionToName[v]; exists {
			return errors.Wrapf(ErrRegistryCollision, "config name=%s conflicts with existing config named=%s", name, n)
		}
		r.versionToName[v] = name
	}
	r.nameToConfig[name] = c
	return nil
}

func (r *registry) Replace(cfg *BeaconChainConfig) error {
	if cfg == nil {
		return ErrReplaceNilConfig
	}
	name := cfg.ConfigName
	r.delete(name)
	if err := r.Add(cfg); err != nil {
		return err
	}
	if r.active.ConfigName == name {
		r.active = cfg
	}
	return nil
}

func (r *registry) delete(name string) {
	c, exists := r.nameToConfig[name]
	if !exists {
		return
	}
	for v, _ := range c.ForkVersionSchedule {
		delete(r.versionToName, v)
	}
	delete(r.nameToConfig, name)
}

func (r *registry) ReplaceWithUndo(cfg *BeaconChainConfig) (func() error, error) {
	name := cfg.ConfigName
	prev := r.nameToConfig[name]
	if err := r.Replace(cfg); err != nil {
		return nil, err
	}
	return func() error {
		if prev == nil {
			if r.active.ConfigName == name {
				return errors.Wrapf(ErrCannotNullifyActive, "active config name=%s", name)
			}
			r.delete(name)
			return nil
		}
		return r.Replace(prev)
	}, nil
}

func (r *registry) GetByName(name string) (*BeaconChainConfig, error) {
	c, ok := r.nameToConfig[name]
	if !ok {
		return nil, errors.Wrapf(ErrConfigNotFound, "name=%s is not a known BeaconChainConfig name", name)
	}
	return c, nil
}

func (r *registry) GetByVersion(version [fieldparams.VersionLength]byte) (*BeaconChainConfig, error) {
	name, ok := r.versionToName[version]
	if !ok {
		return nil, errors.Wrapf(ErrConfigNotFound, "version=%#x not found in any known fork choice schedule", version)
	}
	return r.GetByName(name)
}

func (r *registry) GetActive() *BeaconChainConfig {
	return r.active
}

func (r *registry) SetActive(c *BeaconChainConfig) error {
	r.active = c
	return r.Replace(c)
}

func (r *registry) SetActiveWithUndo(c *BeaconChainConfig) error {

}

func init() {
	defaults := []*BeaconChainConfig{
		MainnetConfig(),
		PraterConfig(),
		MinimalSpecConfig(),
		E2ETestConfig(),
		E2EMainnetTestConfig(),
	}
	Registry = NewRegistry(defaults...)
	// make sure mainnet is present and active
	m, err := Registry.GetByName(MainnetName)
	if err != nil {
		panic(err)
	}
	if Registry.GetActive() != m {
		panic("mainnet should always be the active config at init() time")
	}
}
