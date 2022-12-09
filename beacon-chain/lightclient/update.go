package lightclient

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware/helpers"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
)

// Update is an interface that exposes the attributes common to all types of updates
type Update interface {
	GetAttestedHeader() *ethpbv1.BeaconBlockHeader
	GetSyncAggregate() *ethpbv1.SyncAggregate
	GetSignatureSlot() types.Slot
}

var _ Update = (*ethpbv2.LightClientUpdate)(nil)
var _ Update = (*ethpbv2.LightClientFinalityUpdate)(nil)
var _ Update = (*ethpbv2.LightClientOptimisticUpdate)(nil)

type update struct {
	config *Config
	*ethpbv2.LightClientUpdate
}

var _ Update = (*update)(nil)

func isEmptyWithLength(bb [][]byte, length uint64) bool {
	if len(bb) == 0 {
		return true
	}
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

func (u *update) isSyncCommiteeUpdate() bool {
	return !isEmptyWithLength(u.GetNextSyncCommitteeBranch(), helpers.NextSyncCommitteeIndex)
}

func (u *update) isFinalityUpdate() bool {
	return !isEmptyWithLength(u.GetFinalityBranch(), helpers.FinalizedRootIndex)
}

func (u *update) hasRelevantSyncCommittee() bool {
	return u.isSyncCommiteeUpdate() &&
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
