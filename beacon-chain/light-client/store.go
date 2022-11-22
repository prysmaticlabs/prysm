// Package light_client implements the light client for the Ethereum 2.0 Beacon Chain.
// It is based on the Altair light client spec at this revision:
// https://github.com/ethereum/consensus-specs/tree/208da34ac4e75337baf79adebf036ab595e39f15/specs/altair/light-client
package light_client

import (
	"bytes"
	"errors"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"math"
)

const (
	finalizedRootIndex        = uint64(105)
	currentSyncCommitteeIndex = uint64(54)
	// TODO: read this from the config
	minSyncCommitteeParticipants = uint64(1)
)

type Store struct {
	finalizedHeader               *ethpbv1.BeaconBlockHeader
	currentSyncCommittee          *ethpbv2.SyncCommittee
	nextSyncCommittee             *ethpbv2.SyncCommittee
	bestValidUpdate               *Update
	optimisticHeader              *ethpbv1.BeaconBlockHeader
	previousMaxActiveParticipants uint64
	currentMaxActiveParticipants  uint64
}

func getSubtreeIndex(index uint64) uint64 {
	return index % uint64(math.Pow(2, float64(floorLog2(index-1))))
}

// NewStore implements initialize_light_client_store from the spec.
func NewStore(trustedBlockRoot [32]byte,
	bootstrap ethpbv2.Bootstrap) *Store {
	bootstrapRoot, err := bootstrap.Header.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(trustedBlockRoot[:], bootstrapRoot[:]) {
		panic("trusted block root does not match bootstrap header")
	}
	v1alpha1Committee := ethpb.SyncCommittee{
		Pubkeys:         bootstrap.CurrentSyncCommittee.GetPubkeys(),
		AggregatePubkey: bootstrap.CurrentSyncCommittee.GetAggregatePubkey(),
	}
	syncCommitteeRoot, err := v1alpha1Committee.HashTreeRoot()
	if !trie.VerifyMerkleProofWithDepth(
		bootstrap.Header.StateRoot,
		syncCommitteeRoot[:],
		getSubtreeIndex(currentSyncCommitteeIndex),
		bootstrap.CurrentSyncCommitteeBranch,
		uint64(floorLog2(currentSyncCommitteeIndex))) {
		panic("current sync committee merkle proof is invalid")
	}
	return &Store{
		finalizedHeader:      bootstrap.Header,
		currentSyncCommittee: bootstrap.CurrentSyncCommittee,
		nextSyncCommittee:    &ethpbv2.SyncCommittee{},
		optimisticHeader:     bootstrap.Header,
	}
}

func (s *Store) isNextSyncCommitteeKnown() bool {
	return s.nextSyncCommittee != &ethpbv2.SyncCommittee{}
}

func (s *Store) getSafetyThreshold() uint64 {
	return uint64(math.Floor(math.Max(float64(s.previousMaxActiveParticipants),
		float64(s.currentMaxActiveParticipants)) / 2))
}

func (s *Store) ValidateUpdate(update *Update,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	// Verify sync committee has sufficient participants
	syncAggregate := update.GetSyncAggregate()
	if syncAggregate.SyncCommitteeBits.Count() < minSyncCommitteeParticipants {
		return errors.New("sync committee does not have sufficient participants")
	}

	// Verify update does not skip a sync committee period
	if !(currentSlot >= update.GetSignatureSlot() &&
		update.GetSignatureSlot() > update.GetAttestedHeader().Slot &&
		update.GetAttestedHeader().Slot >= update.GetFinalizedHeader().Slot) {
		return errors.New("update skips a sync committee period")
	}
	// TODO: resume
	return nil
}

func (s *Store) ApplyUpdate(update *Update) error {
	storePeriod := computeSyncCommitteePeriodAtSlot(s.finalizedHeader.Slot)
	updateFinalizedPeriod := computeSyncCommitteePeriodAtSlot(update.GetFinalizedHeader().Slot)
	if !s.isNextSyncCommitteeKnown() {
		if updateFinalizedPeriod != storePeriod {
			return errors.New("update finalized period does not match store period")
		}
		s.nextSyncCommittee = update.GetNextSyncCommittee()
	} else if updateFinalizedPeriod == storePeriod+1 {
		s.currentSyncCommittee = s.nextSyncCommittee
		s.nextSyncCommittee = update.GetNextSyncCommittee()
		s.previousMaxActiveParticipants = s.currentMaxActiveParticipants
		s.currentMaxActiveParticipants = 0
	}
	if update.GetFinalizedHeader().Slot > s.finalizedHeader.Slot {
		s.finalizedHeader = update.GetFinalizedHeader()
		if s.finalizedHeader.Slot > s.optimisticHeader.Slot {
			s.optimisticHeader = s.finalizedHeader
		}
	}
	return nil
}

func (s *Store) ProcessForceUpdate(update *Update) error {
	// TODO: implement
	panic("not implemented")
}

func (s *Store) processUpdate(update *Update,
	currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	if err := s.ValidateUpdate(update, currentSlot, genesisValidatorsRoot); err != nil {
		return err
	}
	syncCommiteeBits := update.GetSyncAggregate().SyncCommitteeBits

	// Update the best update in case we have to force-update to it if the timeout elapses
	if s.bestValidUpdate == nil || update.IsBetterUpdate(s.bestValidUpdate) {
		s.bestValidUpdate = update
	}

	// Track the maximum number of active participants in the committee signature
	s.currentMaxActiveParticipants = uint64(math.Max(float64(s.currentMaxActiveParticipants),
		float64(syncCommiteeBits.Count())))

	// Update the optimistic header
	if syncCommiteeBits.Count() > s.getSafetyThreshold() && update.GetAttestedHeader().Slot > s.optimisticHeader.Slot {
		s.optimisticHeader = update.GetAttestedHeader()
	}

	// Update finalized header
	updateHasFinalizedNextSyncCommittee := !s.isNextSyncCommitteeKnown() && update.IsSyncCommiteeUpdate() &&
		update.IsFinalityUpdate() && computeSyncCommitteePeriodAtSlot(update.GetFinalizedHeader().
		Slot) == computeSyncCommitteePeriodAtSlot(update.GetAttestedHeader().Slot)
	if syncCommiteeBits.Count()*3 >= syncCommiteeBits.Len()*2 || updateHasFinalizedNextSyncCommittee {
		// Normal update throught 2/3 threshold
		if err := s.ApplyUpdate(update); err != nil {
			return err
		}
		s.bestValidUpdate = nil
	}
	return nil
}

func (s *Store) ProcessFinalityUpdate(finalityUpdate *ethpbv2.FinalityUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	return s.processUpdate(&Update{finalityUpdate}, currentSlot, genesisValidatorsRoot)
}

func (s *Store) ProcessOptimisticUpdate(update *ethpbv2.OptimisticUpdate) error {
	// TODO: implement
	panic("not implemented")
}
