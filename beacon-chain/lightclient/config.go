package lightclient

import (
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
	CapellaForkEpoch             types.Epoch `json:"capella_fork_epoch"`
	CapellaForkVersion           []byte      `json:"capella_fork_version"`
	BellatrixForkEpoch           types.Epoch `json:"bellatrix_fork_epoch"`
	BellatrixForkVersion         []byte      `json:"bellatrix_fork_version"`
	AltairForkEpoch              types.Epoch `json:"altair_fork_epoch"`
	AltairForkVersion            []byte      `json:"altair_fork_version"`
	GenesisForkVersion           []byte      `json:"genesis_fork_version"`
	MinSyncCommitteeParticipants uint64      `json:"min_sync_committee_participants"`
	GenesisSlot                  types.Slot  `json:"genesis_slot"`
	DomainSyncCommittee          [4]byte     `json:"domain_sync_committee"`
	SlotsPerEpoch                types.Slot  `json:"slots_per_epoch"`
	EpochsPerSyncCommitteePeriod types.Epoch `json:"epochs_per_sync_committee_period"`
	SecondsPerSlot               uint64      `json:"seconds_per_slot"`
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

func NewConfigFromJSON(config *ConfigJSON) (*Config, error) {
	capellaForkEpoch, err := strconv.ParseUint(config.CapellaForkEpoch, 10, 64)
	if err != nil {
		return nil, err
	}
	bellatrixForkEpoch, err := strconv.ParseUint(config.CapellaForkEpoch, 10, 64)
	if err != nil {
		return nil, err
	}
	altairForkEpoch, err := strconv.ParseUint(config.AltairForkEpoch, 10, 64)
	if err != nil {
		return nil, err
	}
	minSyncCommitteeParticipants, err := strconv.ParseUint(config.MinSyncCommitteeParticipants, 10, 64)
	if err != nil {
		return nil, err
	}
	genesisSlot, err := strconv.ParseUint(config.GenesisSlot, 10, 64)
	if err != nil {
		return nil, err
	}
	domainSyncCommittee, err := strconv.ParseUint(config.DomainSyncCommittee, 10, 32)
	if err != nil {
		return nil, err
	}
	slotsPerEpoch, err := strconv.ParseUint(config.SlotsPerEpoch, 10, 64)
	if err != nil {
		return nil, err
	}
	epochsPerSyncCommitteePeriod, err := strconv.ParseUint(config.EpochsPerSyncCommitteePeriod, 10, 64)
	if err != nil {
		return nil, err
	}
	secondsPerSlot, err := strconv.ParseUint(config.SecondsPerSlot, 10, 64)
	return &Config{
		CapellaForkEpoch:             types.Epoch(capellaForkEpoch),
		CapellaForkVersion:           hexutil.MustDecode(config.CapellaForkVersion),
		BellatrixForkEpoch:           types.Epoch(bellatrixForkEpoch),
		BellatrixForkVersion:         hexutil.MustDecode(config.BellatrixForkVersion),
		AltairForkEpoch:              types.Epoch(altairForkEpoch),
		AltairForkVersion:            hexutil.MustDecode(config.AltairForkVersion),
		GenesisForkVersion:           hexutil.MustDecode(config.GenesisForkVersion),
		MinSyncCommitteeParticipants: minSyncCommitteeParticipants,
		GenesisSlot:                  types.Slot(genesisSlot),
		DomainSyncCommittee:          bytesutil.Uint32ToBytes4(uint32(domainSyncCommittee)),
		SlotsPerEpoch:                types.Slot(slotsPerEpoch),
		EpochsPerSyncCommitteePeriod: types.Epoch(epochsPerSyncCommitteePeriod),
		SecondsPerSlot:               secondsPerSlot,
	}, nil
}
