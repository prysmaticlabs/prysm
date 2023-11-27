package lightclient

import (
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
)

// update is a convenience wrapper for a LightClientUpdate to feed config parameters into misc utils.
type update struct {
	config *Config
	*ethpbv2.LightClientUpdate
}

// hasRelevantSyncCommittee implements has_relevant_sync_committee from the spec.
func (u *update) hasRelevantSyncCommittee() bool {
	return u.IsSyncCommiteeUpdate() &&
		computeSyncCommitteePeriodAtSlot(u.config, u.AttestedHeader.Slot) ==
			computeSyncCommitteePeriodAtSlot(u.config, u.SignatureSlot)
}

// hasSyncCommitteeFinality implements has_sync_committee_finality from the spec.
func (u *update) hasSyncCommitteeFinality() bool {
	return computeSyncCommitteePeriodAtSlot(u.config, u.FinalizedHeader.Slot) ==
		computeSyncCommitteePeriodAtSlot(u.config, u.AttestedHeader.Slot)
}

// isBetterUpdate implements is_better_update from the spec.
func (u *update) isBetterUpdate(newUpdatePb *ethpbv2.LightClientUpdate) bool {
	newUpdate := &update{
		config:            u.config,
		LightClientUpdate: newUpdatePb,
	}
	// Compare supermajority (> 2/3) sync committee participation
	maxActiveParticipants := newUpdate.SyncAggregate.SyncCommitteeBits.Len()
	newNumActiveParticipants := newUpdate.SyncAggregate.SyncCommitteeBits.Count()
	oldNumActiveParticipants := u.SyncAggregate.SyncCommitteeBits.Count()
	newHasSupermajority := newNumActiveParticipants*3 >= maxActiveParticipants*2
	oldHasSupermajority := oldNumActiveParticipants*3 >= maxActiveParticipants*2
	if newHasSupermajority != oldHasSupermajority {
		return newHasSupermajority && !oldHasSupermajority
	}
	if !newHasSupermajority && newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants
	}

	// Compare presence of relevant sync committee
	newHasRelevantSyncCommittee := newUpdate.hasRelevantSyncCommittee()
	oldHasRelevantSyncCommittee := u.hasRelevantSyncCommittee()
	if newHasRelevantSyncCommittee != oldHasRelevantSyncCommittee {
		return newHasRelevantSyncCommittee
	}

	// Compare indication of any finality
	newHasFinality := newUpdate.IsFinalityUpdate()
	oldHasFinality := u.IsFinalityUpdate()
	if newHasFinality != oldHasFinality {
		return newHasFinality
	}

	// Compare sync committee finality
	if newHasFinality {
		newHasSyncCommitteeFinality := newUpdate.hasSyncCommitteeFinality()
		oldHasSyncCommitteeFinality := u.hasSyncCommitteeFinality()
		if newHasSyncCommitteeFinality != oldHasSyncCommitteeFinality {
			return newHasSyncCommitteeFinality
		}
	}

	// Tiebreaker 1: Sync committee participation beyond supermajority
	if newNumActiveParticipants != oldNumActiveParticipants {
		return newNumActiveParticipants > oldNumActiveParticipants
	}

	// Tiebreaker 2: Prefer older data (fewer changes to best)
	if newUpdate.AttestedHeader.Slot != u.AttestedHeader.Slot {
		return newUpdate.AttestedHeader.Slot < u.AttestedHeader.Slot
	}
	return newUpdate.SignatureSlot < u.SignatureSlot
}
