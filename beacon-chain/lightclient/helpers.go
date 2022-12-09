package lightclient

import (
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

func computeEpochAtSlot(config *Config, slot types.Slot) types.Epoch {
	return types.Epoch(slot / config.SlotsPerEpoch)
}

func computeSyncCommitteePeriod(config *Config, epoch types.Epoch) uint64 {
	return uint64(epoch / config.EpochsPerSyncCommitteePeriod)
}

func computeSyncCommitteePeriodAtSlot(config *Config, slot types.Slot) uint64 {
	return computeSyncCommitteePeriod(config, computeEpochAtSlot(config, slot))
}
