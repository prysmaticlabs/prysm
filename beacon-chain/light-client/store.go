// Package light_client implements the light client for the Ethereum 2.0 Beacon Chain.
// It is based on the Altair light client spec at this revision:
// https://github.com/ethereum/consensus-specs/tree/208da34ac4e75337baf79adebf036ab595e39f15/specs/altair/light-client
package light_client

import (
	"bytes"
	"errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/common"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"math"
)

const (
	currentSyncCommitteeIndex = uint64(54)
	altairForkEpoch           = 74240
	// TODO: read these from the config
	minSyncCommitteeParticipants = uint64(1)
	genesisSlot                  = types.Slot(0)
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
	return index % uint64(math.Pow(2, float64(ethpbv2.FloorLog2(index-1))))
}

// TODO: this should be in the proto
func hashTreeRoot(committee *ethpbv2.SyncCommittee) []byte {
	v1alpha1Committee := ethpb.SyncCommittee{
		Pubkeys:         committee.GetPubkeys(),
		AggregatePubkey: committee.GetAggregatePubkey(),
	}
	root, err := v1alpha1Committee.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	return root[:]
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
	if !trie.VerifyMerkleProofWithDepth(
		bootstrap.Header.StateRoot,
		hashTreeRoot(bootstrap.CurrentSyncCommittee),
		getSubtreeIndex(currentSyncCommitteeIndex),
		bootstrap.CurrentSyncCommitteeBranch,
		uint64(ethpbv2.FloorLog2(currentSyncCommitteeIndex))) {
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

func computeForkVersion(epoch uint64) [4]byte {
	// TODO: export these
	altairForkVersion := bytesutil.Uint32ToBytes4(0x01000000)
	genesisForkVersion := bytesutil.Uint32ToBytes4(0x00000000)
	if epoch >= altairForkEpoch {
		return altairForkVersion
	}
	return genesisForkVersion
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
	storePeriod := computeSyncCommitteePeriodAtSlot(s.finalizedHeader.Slot)
	updateSignaturePeriod := computeSyncCommitteePeriodAtSlot(update.GetSignatureSlot())
	if s.isNextSyncCommitteeKnown() {
		if !(updateSignaturePeriod == storePeriod || updateSignaturePeriod == storePeriod+1) {
			return errors.New("update skips a sync committee period")
		}
	} else {
		if updateSignaturePeriod != storePeriod {
			return errors.New("update skips a sync committee period")
		}
	}

	// Verify update is relevant
	updateAttestedPeriod := computeSyncCommitteePeriodAtSlot(update.GetAttestedHeader().Slot)
	updateHasNextSyncCommittee := !s.isNextSyncCommitteeKnown() && (update.IsSyncCommiteeUpdate() && updateAttestedPeriod == storePeriod)
	if !(update.GetAttestedHeader().Slot > s.finalizedHeader.Slot || updateHasNextSyncCommittee) {
		return errors.New("update is not relevant")
	}

	// Verify that the finality branch, if present, confirms finalized header to match the finalized checkpoint root
	// saved in the state of attested header. Note that the genesis finalized checkpoint root is represented as a zero
	// hash.
	if !update.IsFinalityUpdate() {
		if update.GetFinalizedHeader() != &(ethpbv1.BeaconBlockHeader{}) {
			return errors.New("finality branch is present but update is not finality")
		}
	} else {
		var finalizedRoot [32]byte
		if update.GetFinalizedHeader().Slot == genesisSlot {
			if update.GetFinalizedHeader() != &(ethpbv1.BeaconBlockHeader{}) {
				return errors.New("finality branch is present but update is not finality")
			}
			finalizedRoot = [32]byte{}
		} else {
			var err error
			if finalizedRoot, err = update.GetFinalizedHeader().HashTreeRoot(); err != nil {
				return err
			}
		}
		if !trie.VerifyMerkleProofWithDepth(
			update.GetAttestedHeader().StateRoot,
			finalizedRoot[:],
			getSubtreeIndex(ethpbv2.FinalizedRootIndex),
			update.GetFinalityBranch(),
			uint64(ethpbv2.FloorLog2(ethpbv2.FinalizedRootIndex))) {
			return errors.New("finality branch is invalid")
		}
	}

	// Verify that the next sync committee, if present, actually is the next sync committee saved in the state of the
	// attested header
	if !update.IsSyncCommiteeUpdate() {
		if update.GetNextSyncCommittee() != &(ethpbv2.SyncCommittee{}) {
			return errors.New("sync committee branch is present but update is not sync committee")
		}
	} else {
		if updateAttestedPeriod == storePeriod && s.isNextSyncCommitteeKnown() {
			if update.GetNextSyncCommittee() != s.nextSyncCommittee {
				return errors.New("next sync committee is not known")
			}
		}
		if !trie.VerifyMerkleProofWithDepth(
			update.GetAttestedHeader().StateRoot,
			hashTreeRoot(update.GetNextSyncCommittee())[:],
			getSubtreeIndex(ethpbv2.NextSyncCommitteeIndex),
			update.GetNextSyncCommitteeBranch(),
			uint64(ethpbv2.FloorLog2(ethpbv2.NextSyncCommitteeIndex))) {
			return errors.New("sync committee branch is invalid")
		}
	}

	var syncCommittee *ethpbv2.SyncCommittee
	// Verify sync committee aggregate signature
	if updateSignaturePeriod == storePeriod {
		syncCommittee = s.currentSyncCommittee
	} else {
		syncCommittee = s.nextSyncCommittee
	}
	participantPubkeys := []common.PublicKey{}
	for i, bit := range syncAggregate.SyncCommitteeBits {
		if bit > 0 {
			publicKey, err := blst.PublicKeyFromBytes(syncCommittee.Pubkeys[i])
			if err != nil {
				return err
			}
			participantPubkeys = append(participantPubkeys, publicKey)
		}
	}
	forkVersion := computeForkVersion(computeEpochAtSlot(update.GetSignatureSlot()))
	// TODO: export this somewhere
	domainSyncCommittee := bytesutil.Uint32ToBytes4(0x07000000)
	domain, err := signing.ComputeDomain(domainSyncCommittee, forkVersion[:], genesisValidatorsRoot)
	if err != nil {
		return err
	}
	signingRoot, err := signing.ComputeSigningRoot(update.GetAttestedHeader(), domain)
	if err != nil {
		return err
	}
	signature, err := blst.SignatureFromBytes(syncAggregate.SyncCommitteeSignature)
	if err != nil {
		return err
	}
	if !signature.FastAggregateVerify(participantPubkeys, signingRoot) {
		return errors.New("sync committee signature is invalid")
	}
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

func (s *Store) ProcessOptimisticUpdate(optimisticUpdate *ethpbv2.OptimisticUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	// TODO: implement
	return s.processUpdate(&Update{optimisticUpdate}, currentSlot, genesisValidatorsRoot)
}
