package params

import (
	"encoding/hex"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/math"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type CombinedConfig struct {
	BeaconChainConfig *BeaconChainConfig
	NetworkConfig     *NetworkConfig
}

var errNilConfig = errors.New("config must not be nil when unmarshalling from yaml")

func (c *CombinedConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// each config struct should be set at this point, to avoid zero values
	if c.BeaconChainConfig == nil {
		return errors.Wrap(errNilConfig, "chain config")
	}
	if err := unmarshal(c.BeaconChainConfig); err != nil {
		return err
	}
	if c.NetworkConfig == nil {
		return errors.Wrap(errNilConfig, "network config")
	}
	return unmarshal(c.NetworkConfig)
}

func isMinimal(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, "PRESET_BASE: 'minimal'") ||
			strings.HasPrefix(l, `PRESET_BASE: "minimal"`) ||
			strings.HasPrefix(l, "PRESET_BASE: minimal") ||
			strings.HasPrefix(l, "# Minimal preset") {
			return true
		}
	}
	return false
}

func UnmarshalConfig(yamlFile []byte, conf *BeaconChainConfig) (*CombinedConfig, error) {
	// To track if config name is defined inside config file.
	hasConfigName := false
	// Convert 0x hex inputs to fixed bytes arrays
	lines := strings.Split(string(yamlFile), "\n")
	if conf == nil {
		if isMinimal(lines) {
			conf = MinimalSpecConfig().Copy()
		} else {
			// Default to using mainnet.
			conf = MainnetConfig().Copy()
		}
	}
	for i, line := range lines {
		// No need to convert the deposit contract address to byte array (as config expects a string).
		if strings.HasPrefix(line, "DEPOSIT_CONTRACT_ADDRESS") {
			continue
		}
		if strings.HasPrefix(line, "CONFIG_NAME") {
			hasConfigName = true
		}
		if !strings.HasPrefix(line, "#") && strings.Contains(line, "0x") {
			parts := ReplaceHexStringWithYAMLFormat(line)
			lines[i] = strings.Join(parts, "\n")
		}
	}
	yamlFile = []byte(strings.Join(lines, "\n"))
	cconf := &CombinedConfig{
		BeaconChainConfig: conf,
		NetworkConfig:     BeaconNetworkConfig(),
	}
	if err := yaml.Unmarshal(yamlFile, cconf); err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return nil, errors.Wrap(err, "Failed to parse config yaml file.")
		} else {
			log.WithError(err).Error("There were some issues parsing the config from a yaml file")
		}
	}
	if !hasConfigName {
		cconf.BeaconChainConfig.ConfigName = DevnetName
	}
	// recompute SqrRootSlotsPerEpoch constant to handle non-standard values of SlotsPerEpoch
	cconf.BeaconChainConfig.SqrRootSlotsPerEpoch = primitives.Slot(math.IntegerSquareRoot(uint64(cconf.BeaconChainConfig.SlotsPerEpoch)))
	log.Debugf("Config file values: %+v", cconf)
	return cconf, nil
}

func UnmarshalConfigFile(path string, conf *BeaconChainConfig) (*CombinedConfig, error) {
	yamlFile, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read chain config file.")
	}
	return UnmarshalConfig(yamlFile, conf)
}

// LoadChainConfigFile load, convert hex values into valid param yaml format,
// unmarshal , and apply beacon chain config file.
func LoadChainConfigFile(path string, conf *BeaconChainConfig) error {
	c, err := UnmarshalConfigFile(path, conf)
	if err != nil {
		return err
	}
	OverrideBeaconNetworkConfig(c.NetworkConfig)
	return SetActive(c.BeaconChainConfig)
}

// ReplaceHexStringWithYAMLFormat will replace hex strings that the yaml parser will understand.
func ReplaceHexStringWithYAMLFormat(line string) []string {
	parts := strings.Split(line, "0x")
	decoded, err := hex.DecodeString(parts[1])
	if err != nil {
		log.WithError(err).Error("Failed to decode hex string.")
	}
	switch l := len(decoded); {
	case l == 1:
		var b byte
		b = decoded[0]
		fixedByte, err := yaml.Marshal(b)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[0] += string(fixedByte)
		parts = parts[:1]
	case l > 1 && l <= 4:
		var arr [4]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 4 && l <= 8:
		var arr [8]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 8 && l <= 16:
		var arr [16]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 16 && l <= 20:
		var arr [20]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 20 && l <= 32:
		var arr [32]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 32 && l <= 48:
		var arr [48]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 48 && l <= 64:
		var arr [64]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	case l > 64 && l <= 96:
		var arr [96]byte
		copy(arr[:], decoded)
		fixedByte, err := yaml.Marshal(arr)
		if err != nil {
			log.WithError(err).Error("Failed to marshal config file.")
		}
		parts[1] = string(fixedByte)
	}
	return parts
}
