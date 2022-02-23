package params

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/math"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// LoadChainConfigFile load, convert hex values into valid param yaml format,
// unmarshal , and apply beacon chain config file.
func LoadChainConfigFile(chainConfigFileName string) {
	yamlFile, err := ioutil.ReadFile(chainConfigFileName) // #nosec G304
	if err != nil {
		log.WithError(err).Fatal("Failed to read chain config file.")
	}
	// Default to using mainnet.
	conf := MainnetConfig().Copy()
	// To track if config name is defined inside config file.
	hasConfigName := false
	// Convert 0x hex inputs to fixed bytes arrays
	lines := strings.Split(string(yamlFile), "\n")
	for i, line := range lines {
		// No need to convert the deposit contract address to byte array (as config expects a string).
		if strings.HasPrefix(line, "DEPOSIT_CONTRACT_ADDRESS") {
			continue
		}
		if strings.HasPrefix(line, "CONFIG_NAME") {
			hasConfigName = true
		}
		if strings.HasPrefix(line, "PRESET_BASE: 'minimal'") ||
			strings.HasPrefix(line, `PRESET_BASE: "minimal"`) ||
			strings.HasPrefix(line, "PRESET_BASE: minimal") ||
			strings.HasPrefix(line, "# Minimal preset") {
			conf = MinimalSpecConfig().Copy()
		}
		if !strings.HasPrefix(line, "#") && strings.Contains(line, "0x") {
			parts := ReplaceHexStringWithYAMLFormat(line)
			lines[i] = strings.Join(parts, "\n")
		}
	}
	yamlFile = []byte(strings.Join(lines, "\n"))
	if err := yaml.UnmarshalStrict(yamlFile, conf); err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			log.WithError(err).Fatal("Failed to parse chain config yaml file.")
		} else {
			log.WithError(err).Error("There were some issues parsing the config from a yaml file")
		}
	}
	if !hasConfigName {
		conf.ConfigName = "devnet"
	}
	// recompute SqrRootSlotsPerEpoch constant to handle non-standard values of SlotsPerEpoch
	conf.SqrRootSlotsPerEpoch = types.Slot(math.IntegerSquareRoot(uint64(conf.SlotsPerEpoch)))
	log.Debugf("Config file values: %+v", conf)
	OverrideBeaconConfig(conf)
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

// ConfigToYaml takes a provided config and outputs its contents
// in yaml. This allows prysm's custom configs to be read by other clients.
func ConfigToYaml(cfg *BeaconChainConfig) []byte {
	lines := []string{}
	lines = append(lines, fmt.Sprintf("PRESET_BASE: '%s'", cfg.PresetBase))
	lines = append(lines, fmt.Sprintf("CONFIG_NAME: '%s'", cfg.ConfigName))
	lines = append(lines, fmt.Sprintf("MIN_GENESIS_ACTIVE_VALIDATOR_COUNT: %d", cfg.MinGenesisActiveValidatorCount))
	lines = append(lines, fmt.Sprintf("GENESIS_DELAY: %d", cfg.GenesisDelay))
	lines = append(lines, fmt.Sprintf("MIN_GENESIS_TIME: %d", cfg.MinGenesisTime))
	lines = append(lines, fmt.Sprintf("GENESIS_FORK_VERSION: %#x", cfg.GenesisForkVersion))
	lines = append(lines, fmt.Sprintf("CHURN_LIMIT_QUOTIENT: %d", cfg.ChurnLimitQuotient))
	lines = append(lines, fmt.Sprintf("SECONDS_PER_SLOT: %d", cfg.SecondsPerSlot))
	lines = append(lines, fmt.Sprintf("SLOTS_PER_EPOCH: %d", cfg.SlotsPerEpoch))
	lines = append(lines, fmt.Sprintf("SECONDS_PER_ETH1_BLOCK: %d", cfg.SecondsPerETH1Block))
	lines = append(lines, fmt.Sprintf("ETH1_FOLLOW_DISTANCE: %d", cfg.Eth1FollowDistance))
	lines = append(lines, fmt.Sprintf("EPOCHS_PER_ETH1_VOTING_PERIOD: %d", cfg.EpochsPerEth1VotingPeriod))
	lines = append(lines, fmt.Sprintf("SHARD_COMMITTEE_PERIOD: %d", cfg.ShardCommitteePeriod))
	lines = append(lines, fmt.Sprintf("MIN_VALIDATOR_WITHDRAWABILITY_DELAY: %d", cfg.MinValidatorWithdrawabilityDelay))
	lines = append(lines, fmt.Sprintf("MAX_SEED_LOOKAHEAD: %d", cfg.MaxSeedLookahead))
	lines = append(lines, fmt.Sprintf("EJECTION_BALANCE: %d", cfg.EjectionBalance))
	lines = append(lines, fmt.Sprintf("MIN_PER_EPOCH_CHURN_LIMIT: %d", cfg.MinPerEpochChurnLimit))
	lines = append(lines, fmt.Sprintf("DEPOSIT_CHAIN_ID: %d", cfg.DepositChainID))
	lines = append(lines, fmt.Sprintf("DEPOSIT_NETWORK_ID: %d", cfg.DepositNetworkID))
	lines = append(lines, fmt.Sprintf("ALTAIR_FORK_EPOCH: %d", cfg.AltairForkEpoch))
	lines = append(lines, fmt.Sprintf("ALTAIR_FORK_VERSION: %#x", cfg.AltairForkVersion))
	lines = append(lines, fmt.Sprintf("INACTIVITY_SCORE_BIAS: %d", cfg.InactivityScoreBias))
	lines = append(lines, fmt.Sprintf("INACTIVITY_SCORE_RECOVERY_RATE: %d", cfg.InactivityScoreRecoveryRate))
	lines = append(lines, fmt.Sprintf("TERMINAL_TOTAL_DIFFICULTY: %s", cfg.TerminalTotalDifficulty))
	lines = append(lines, fmt.Sprintf("TERMINAL_BLOCK_HASH: %#x", cfg.TerminalBlockHash))
	lines = append(lines, fmt.Sprintf("TERMINAL_BLOCK_HASH_ACTIVATION_EPOCH: %d", cfg.TerminalBlockHashActivationEpoch))

	yamlFile := []byte(strings.Join(lines, "\n"))
	return yamlFile
}
