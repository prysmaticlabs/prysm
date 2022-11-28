package light_client

import (
	"bytes"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

const (
	// TODO: we should read these from the beacon chain config
	epochsPerSyncCommitteePeriod = uint64(256)
	slotsPerEpoch                = types.Slot(32)
)

type Update struct {
	ethpbv2.Update `json:"update,omitempty"`
}

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	l := ethpbv2.FloorLog2(length)
	if len(bb) != l {
		return false
	}
	for _, b := range bb {
		if !bytes.Equal(b, []byte{}) {
			return false
		}
	}
	return true
}

func computeEpochAtSlot(slot types.Slot) types.Epoch {
	return types.Epoch(slot / slotsPerEpoch)
}

func computeSyncCommitteePeriod(epoch types.Epoch) uint64 {
	return uint64(epoch) / epochsPerSyncCommitteePeriod
}

func computeSyncCommitteePeriodAtSlot(slot types.Slot) uint64 {
	return computeSyncCommitteePeriod(computeEpochAtSlot(slot))
}

func (u *Update) isSyncCommiteeUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), ethpbv2.NextSyncCommitteeIndex)
}

func (u *Update) isFinalityUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), ethpbv2.FinalizedRootIndex)
}

func (u *Update) hasRelevantSyncCommittee() bool {
	return u.isSyncCommiteeUpdate() &&
		computeSyncCommitteePeriodAtSlot(u.GetAttestedHeader().Slot) == computeSyncCommitteePeriodAtSlot(u.GetSignatureSlot())
}

func (u *Update) hasSyncCommitteeFinality() bool {
	return computeSyncCommitteePeriodAtSlot(u.GetFinalizedHeader().Slot) == computeSyncCommitteePeriodAtSlot(u.
		GetAttestedHeader().Slot)
}

func (u *Update) isBetterUpdate(newUpdate *Update) bool {
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
	newHasFinality := newUpdate.isFinalityUpdate()
	oldHasFinality := u.isFinalityUpdate()
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
