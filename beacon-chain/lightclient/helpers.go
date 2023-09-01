package lightclient

import (
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
)

// computeEpochAtSlot implements compute_epoch_at_slot from the spec.
func computeEpochAtSlot(config *Config, slot types.Slot) types.Epoch {
	return types.Epoch(slot / config.SlotsPerEpoch)
}

// computeSyncCommitteePeriod implements compute_sync_committee_period from the spec.
func computeSyncCommitteePeriod(config *Config, epoch types.Epoch) uint64 {
	return uint64(epoch / config.EpochsPerSyncCommitteePeriod)
}

// computeSyncCommitteePeriodAtSlot implements compute_sync_committee_period_at_slot from the spec.
func computeSyncCommitteePeriodAtSlot(config *Config, slot types.Slot) uint64 {
	return computeSyncCommitteePeriod(config, computeEpochAtSlot(config, slot))
}
