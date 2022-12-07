package lightclient

import (
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

type Config struct {
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
