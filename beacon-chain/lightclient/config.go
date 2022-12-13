package lightclient

import (
	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
)

type ConfigJSON struct {
	CapellaForkEpoch             string `json:"capella_fork_epoch"`
	CapellaForkVersion           string `json:"capella_fork_version" hex:"true"`
	BellatrixForkEpoch           string `json:"bellatrix_fork_epoch"`
	BellatrixForkVersion         string `json:"bellatrix_fork_version" hex:"true"`
	AltairForkEpoch              string `json:"altair_fork_epoch"`
	AltairForkVersion            string `json:"altair_fork_version" hex:"true"`
	GenesisForkVersion           string `json:"genesis_fork_version" hex:"true"`
	MinSyncCommitteeParticipants string `json:"min_sync_committee_participants"`
	GenesisSlot                  string `json:"genesis_slot"`
	DomainSyncCommittee          string `json:"domain_sync_committee" hex:"true"`
	SlotsPerEpoch                string `json:"slots_per_epoch"`
	EpochsPerSyncCommitteePeriod string `json:"epochs_per_sync_committee_period"`
	SecondsPerSlot               string `json:"seconds_per_slot"`
}

type Config struct {
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

func NewConfig(chainConfig *params.BeaconChainConfig) *Config {
	return &Config{
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
	var config ConfigJSON
	if err := json.Unmarshal(input, &config); err != nil {
		return err
	}
	capellaForkEpoch, err := strconv.ParseUint(config.CapellaForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	bellatrixForkEpoch, err := strconv.ParseUint(config.CapellaForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	altairForkEpoch, err := strconv.ParseUint(config.AltairForkEpoch, 10, 64)
	if err != nil {
		return err
	}
	minSyncCommitteeParticipants, err := strconv.ParseUint(config.MinSyncCommitteeParticipants, 10, 64)
	if err != nil {
		return err
	}
	genesisSlot, err := strconv.ParseUint(config.GenesisSlot, 10, 64)
	if err != nil {
		return err
	}
	slotsPerEpoch, err := strconv.ParseUint(config.SlotsPerEpoch, 10, 64)
	if err != nil {
		return err
	}
	epochsPerSyncCommitteePeriod, err := strconv.ParseUint(config.EpochsPerSyncCommitteePeriod, 10, 64)
	if err != nil {
		return err
	}
	secondsPerSlot, err := strconv.ParseUint(config.SecondsPerSlot, 10, 64)
	c = &Config{
		CapellaForkEpoch:             types.Epoch(capellaForkEpoch),
		CapellaForkVersion:           hexutil.MustDecode(config.CapellaForkVersion),
		BellatrixForkEpoch:           types.Epoch(bellatrixForkEpoch),
		BellatrixForkVersion:         hexutil.MustDecode(config.BellatrixForkVersion),
		AltairForkEpoch:              types.Epoch(altairForkEpoch),
		AltairForkVersion:            hexutil.MustDecode(config.AltairForkVersion),
		GenesisForkVersion:           hexutil.MustDecode(config.GenesisForkVersion),
		MinSyncCommitteeParticipants: minSyncCommitteeParticipants,
		GenesisSlot:                  types.Slot(genesisSlot),
		DomainSyncCommittee:          bytesutil.ToBytes4(hexutil.MustDecode(config.DomainSyncCommittee)),
		SlotsPerEpoch:                types.Slot(slotsPerEpoch),
		EpochsPerSyncCommitteePeriod: types.Epoch(epochsPerSyncCommitteePeriod),
		SecondsPerSlot:               secondsPerSlot,
	}
	return nil
}
