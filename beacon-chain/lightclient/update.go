package lightclient

import (
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

type update struct {
	config *Config
	*ethpbv2.LightClientUpdate
}

func (u *update) hasRelevantSyncCommittee() bool {
	return u.IsSyncCommiteeUpdate() &&
		computeSyncCommitteePeriodAtSlot(u.config, u.GetAttestedHeader().Slot) ==
			computeSyncCommitteePeriodAtSlot(u.config, u.GetSignatureSlot())
}

func (u *update) hasSyncCommitteeFinality() bool {
	return computeSyncCommitteePeriodAtSlot(u.config, u.GetFinalizedHeader().Slot) ==
		computeSyncCommitteePeriodAtSlot(u.config, u.GetAttestedHeader().Slot)
}

func (u *update) isBetterUpdate(newUpdatePb *ethpbv2.LightClientUpdate) bool {
	newUpdate := &update{
		config:            u.config,
		LightClientUpdate: newUpdatePb,
	}
	// Compare supermajority (> 2/3) sync committee participation
	maxActiveParticipants := newUpdate.GetSyncAggregate().SyncCommitteeBits.Len()
	newNumActiveParticipants := newUpdate.GetSyncAggregate().SyncCommitteeBits.Count()
	oldNumActiveParticipants := u.GetSyncAggregate().SyncCommitteeBits.Count()
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
	if newUpdate.GetAttestedHeader().Slot != u.GetAttestedHeader().Slot {
		return newUpdate.GetAttestedHeader().Slot < u.GetAttestedHeader().Slot
	}
	return newUpdate.GetSignatureSlot() < u.GetSignatureSlot()
}
