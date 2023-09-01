package lightclient

import (
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
)

// ConfigJSON is the JSON representation of the light client config.
type ConfigJSON struct {
	DenebForkEpoch               string `json:"deneb_fork_epoch"`
	DenebForkVersion             string `json:"deneb_fork_version"               hex:"true"`
	CapellaForkEpoch             string `json:"capella_fork_epoch"`
	CapellaForkVersion           string `json:"capella_fork_version"             hex:"true"`
	BellatrixForkEpoch           string `json:"bellatrix_fork_epoch"`
	BellatrixForkVersion         string `json:"bellatrix_fork_version"           hex:"true"`
	AltairForkEpoch              string `json:"altair_fork_epoch"`
	AltairForkVersion            string `json:"altair_fork_version"              hex:"true"`
	GenesisForkVersion           string `json:"genesis_fork_version"             hex:"true"`
	MinSyncCommitteeParticipants string `json:"min_sync_committee_participants"`
	GenesisSlot                  string `json:"genesis_slot"`
	DomainSyncCommittee          string `json:"domain_sync_committee"            hex:"true"`
	SlotsPerEpoch                string `json:"slots_per_epoch"`
	EpochsPerSyncCommitteePeriod string `json:"epochs_per_sync_committee_period"`
	SecondsPerSlot               string `json:"seconds_per_slot"`
}

// Config is the light client configuration. It consists of the subset of the beacon chain configuration relevant to the
// light client. Unlike the beacon chain configuration it is serializable to JSON, hence it's a separate object.
type Config struct {
	DenebForkEpoch               types.Epoch
	DenebForkVersion             []byte
	CapellaForkEpoch             types.Epoch
	CapellaForkVersion           []byte
	BellatrixForkEpoch           types.Epoch
	BellatrixForkVersion         []byte
	AltairForkEpoch              types.Epoch
	AltairForkVersion            []byte
	GenesisForkVersion           []byte
	MinSyncCommitteeParticipants uint64
	GenesisSlot                  types.Slot
	DomainSyncCommittee          [4]byte
	SlotsPerEpoch                types.Slot
	EpochsPerSyncCommitteePeriod types.Epoch
	SecondsPerSlot               uint64
}

// NewConfig creates a new light client configuration from a beacon chain configuration.
func NewConfig(chainConfig *params.BeaconChainConfig) *Config {
	return &Config{
		DenebForkEpoch:               chainConfig.DenebForkEpoch,
		DenebForkVersion:             chainConfig.DenebForkVersion,
		CapellaForkEpoch:             chainConfig.CapellaForkEpoch,
		CapellaForkVersion:           chainConfig.CapellaForkVersion,
		BellatrixForkEpoch:           chainConfig.BellatrixForkEpoch,
		BellatrixForkVersion:         chainConfig.BellatrixForkVersion,
		AltairForkEpoch:              chainConfig.AltairForkEpoch,
		AltairForkVersion:            chainConfig.AltairForkVersion,
		GenesisForkVersion:           chainConfig.GenesisForkVersion,
		MinSyncCommitteeParticipants: chainConfig.MinSyncCommitteeParticipants,
		GenesisSlot:                  chainConfig.GenesisSlot,
		DomainSyncCommittee:          chainConfig.DomainSyncCommittee,
		SlotsPerEpoch:                chainConfig.SlotsPerEpoch,
		EpochsPerSyncCommitteePeriod: chainConfig.EpochsPerSyncCommitteePeriod,
		SecondsPerSlot:               chainConfig.SecondsPerSlot,
	}
}

func (c *Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(&ConfigJSON{
		DenebForkEpoch:               strconv.FormatUint(uint64(c.DenebForkEpoch), 10),
		DenebForkVersion:             hexutil.Encode(c.DenebForkVersion),
		CapellaForkEpoch:             strconv.FormatUint(uint64(c.CapellaForkEpoch), 10),
		CapellaForkVersion:           hexutil.Encode(c.CapellaForkVersion),
		BellatrixForkEpoch:           strconv.FormatUint(uint64(c.BellatrixForkEpoch), 10),
		BellatrixForkVersion:         hexutil.Encode(c.BellatrixForkVersion),
		AltairForkEpoch:              strconv.FormatUint(uint64(c.AltairForkEpoch), 10),
		AltairForkVersion:            hexutil.Encode(c.AltairForkVersion),
		GenesisForkVersion:           hexutil.Encode(c.GenesisForkVersion),
		MinSyncCommitteeParticipants: strconv.FormatUint(c.MinSyncCommitteeParticipants, 10),
		GenesisSlot:                  strconv.FormatUint(uint64(c.GenesisSlot), 10),
		DomainSyncCommittee:          hexutil.Encode(c.DomainSyncCommittee[:]),
		SlotsPerEpoch:                strconv.FormatUint(uint64(c.SlotsPerEpoch), 10),
		EpochsPerSyncCommitteePeriod: strconv.FormatUint(uint64(c.EpochsPerSyncCommitteePeriod), 10),
		SecondsPerSlot:               strconv.FormatUint(c.SecondsPerSlot, 10),
	})
}

func (c *Config) UnmarshalJSON(input []byte) error {
	var configJSON ConfigJSON
	if err := json.Unmarshal(input, &configJSON); err != nil {
		return err
	}
	var config Config

	denebForkEpoch, err := strconv.ParseUint(configJSON.DenebForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	config.DenebForkEpoch = types.Epoch(denebForkEpoch)
	if config.DenebForkVersion, err = hexutil.Decode(configJSON.DenebForkVersion); err != nil {
		return err
	}
	capellaForkEpoch, err := strconv.ParseUint(configJSON.CapellaForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	config.CapellaForkEpoch = types.Epoch(capellaForkEpoch)
	if config.CapellaForkVersion, err = hexutil.Decode(configJSON.CapellaForkVersion); err != nil {
		return err
	}
	bellatrixForkEpoch, err := strconv.ParseUint(configJSON.BellatrixForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	config.BellatrixForkEpoch = types.Epoch(bellatrixForkEpoch)
	if config.BellatrixForkVersion, err = hexutil.Decode(configJSON.BellatrixForkVersion); err != nil {
		return err
	}
	altairForkEpoch, err := strconv.ParseUint(configJSON.AltairForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	config.AltairForkEpoch = types.Epoch(altairForkEpoch)
	if config.AltairForkVersion, err = hexutil.Decode(configJSON.AltairForkVersion); err != nil {
		return err
	}
	if config.GenesisForkVersion, err = hexutil.Decode(configJSON.GenesisForkVersion); err != nil {
		return err
	}
	if config.MinSyncCommitteeParticipants, err = strconv.ParseUint(configJSON.MinSyncCommitteeParticipants, 10, 64); err != nil {
		return err
	}
	genesisSlot, err := strconv.ParseUint(configJSON.GenesisSlot, 10, 64)
	if err != nil {
		return err
	}
	config.GenesisSlot = types.Slot(genesisSlot)
	domainSyncCommittee, err := hexutil.Decode(configJSON.DomainSyncCommittee)
	if err != nil {
		return err
	}
	config.DomainSyncCommittee = bytesutil.ToBytes4(domainSyncCommittee)
	slotsPerEpoch, err := strconv.ParseUint(configJSON.SlotsPerEpoch, 10, 64)
	if err != nil {
		return err
	}
	config.SlotsPerEpoch = types.Slot(slotsPerEpoch)
	epochsPerSyncCommitteePeriod, err := strconv.ParseUint(configJSON.EpochsPerSyncCommitteePeriod, 10, 64)
	if err != nil {
		return err
	}
	config.EpochsPerSyncCommitteePeriod = types.Epoch(epochsPerSyncCommitteePeriod)
	if config.SecondsPerSlot, err = strconv.ParseUint(configJSON.SecondsPerSlot, 10, 64); err != nil {
		return err
	}
	*c = config
	return nil
}
