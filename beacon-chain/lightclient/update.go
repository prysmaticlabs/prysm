package lightclient

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware/helpers"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	l := helpers.FloorLog2(length)
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

type Update struct {
	Config                     *Config
	*ethpbv2.LightClientUpdate `json:"update,omitempty"`
}

func (u *Update) isSyncCommiteeUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), helpers.NextSyncCommitteeIndex)
}

func (u *Update) isFinalityUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), helpers.FinalizedRootIndex)
}

func (u *Update) hasRelevantSyncCommittee() bool {
	return u.isSyncCommiteeUpdate() &&
		computeSyncCommitteePeriodAtSlot(u.Config, u.GetAttestedHeader().Slot) ==
			computeSyncCommitteePeriodAtSlot(u.Config, u.GetSignatureSlot())
}

func (u *Update) hasSyncCommitteeFinality() bool {
	return computeSyncCommitteePeriodAtSlot(u.Config, u.GetFinalizedHeader().Slot) ==
		computeSyncCommitteePeriodAtSlot(u.Config, u.GetAttestedHeader().Slot)
}

func (u *Update) isBetterUpdate(newUpdate *Update) bool {
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
