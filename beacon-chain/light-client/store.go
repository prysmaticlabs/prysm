// Package light_client implements the light client for the Ethereum 2.0 Beacon Chain.
// It is based on the Altair light client spec at this revision:
// https://github.com/ethereum/consensus-specs/tree/208da34ac4e75337baf79adebf036ab595e39f15/specs/altair/light-client
package light_client

import (
	"errors"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/container/trie"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls/common"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

type Store struct {
	// BeaconChainConfig is the config for the beacon chain
	BeaconChainConfig *params.BeaconChainConfig `json:"beacon_chain_config,omitempty"`
	// FinalizedHeader is a header that is finalized
	FinalizedHeader *ethpbv1.BeaconBlockHeader `json:"finalized_header,omitempty"`
	// CurrentSyncCommittee is the sync committees corresponding to the finalized header
	CurrentSyncCommittee *ethpbv2.SyncCommittee `json:"current_sync_committeeu,omitempty"`
	// NextSyncCommittee is the next sync committees corresponding to the finalized header
	NextSyncCommittee *ethpbv2.SyncCommittee `json:"next_sync_committee,omitempty"`
	// BestValidUpdate is the best available header to switch finalized head to if we see nothing else
	BestValidUpdate *Update `json:"best_valid_update,omitempty"`
	// OptimisticHeader os the most recent available reasonably-safe header
	OptimisticHeader *ethpbv1.BeaconBlockHeader `json:"optimistic_header,omitempty"`
	// PreviousMaxActiveParticipants is the previous max number of active participants in a sync committee (used to
	// calculate safety threshold)
	PreviousMaxActiveParticipants uint64 `json:"previous_max_active_participants,omitempty"`
	// CurrentMaxActiveParticipants is the max number of active participants in a sync committee (used to calculate
	// safety threshold)
	CurrentMaxActiveParticipants uint64 `json:"current_max_active_participants,omitempty"`
}

func getSubtreeIndex(index uint64) uint64 {
	return index % (uint64(1) << ethpbv2.FloorLog2(index-1))
}

// TODO: this should be in the proto
func hashTreeRoot(committee *ethpbv2.SyncCommittee) ([]byte, error) {
	v1alpha1Committee := ethpb.SyncCommittee{
		Pubkeys:         committee.GetPubkeys(),
		AggregatePubkey: committee.GetAggregatePubkey(),
	}
	root, err := v1alpha1Committee.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	return root[:], nil
}

// NewStore implements initialize_light_client_store from the spec.
func NewStore(config *params.BeaconChainConfig, trustedBlockRoot [32]byte,
	bootstrap ethpbv2.Bootstrap) (*Store, error) {
	bootstrapRoot, err := bootstrap.Header.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	if trustedBlockRoot == bootstrapRoot {
		panic("trusted block root does not match bootstrap header")
	}
	root, err := hashTreeRoot(bootstrap.CurrentSyncCommittee)
	if err != nil {
		return nil, err
	}
	if !trie.VerifyMerkleProofWithDepth(
		bootstrap.Header.StateRoot,
		root,
		getSubtreeIndex(ethpbv2.CurrentSyncCommitteeIndex),
		bootstrap.CurrentSyncCommitteeBranch,
		uint64(ethpbv2.FloorLog2(ethpbv2.CurrentSyncCommitteeIndex))) {
		panic("current sync committee merkle proof is invalid")
	}
	return &Store{
		BeaconChainConfig:    config,
		FinalizedHeader:      bootstrap.Header,
		CurrentSyncCommittee: bootstrap.CurrentSyncCommittee,
		NextSyncCommittee:    nil,
		OptimisticHeader:     bootstrap.Header,
	}, nil
}

func (s *Store) isNextSyncCommitteeKnown() bool {
	return s.NextSyncCommittee != nil
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func (s *Store) getSafetyThreshold() uint64 {
	return max(s.PreviousMaxActiveParticipants, s.CurrentMaxActiveParticipants) / 2
}

func (s *Store) computeForkVersion(epoch types.Epoch) []byte {
	if epoch >= s.BeaconChainConfig.AltairForkEpoch {
		return s.BeaconChainConfig.AltairForkVersion
	}
	return s.BeaconChainConfig.GenesisForkVersion
}

func (s *Store) validateUpdate(update *Update,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	// Verify sync committee has sufficient participants
	syncAggregate := update.GetSyncAggregate()
	if syncAggregate.SyncCommitteeBits.Count() < s.BeaconChainConfig.MinSyncCommitteeParticipants {
		return errors.New("sync committee does not have sufficient participants")
	}

	// Verify update does not skip a sync committee period
	if !(currentSlot >= update.GetSignatureSlot() &&
		update.GetSignatureSlot() > update.GetAttestedHeader().Slot &&
		update.GetAttestedHeader().Slot >= update.GetFinalizedHeader().Slot) {
		return errors.New("update skips a sync committee period")
	}
	storePeriod := update.computeSyncCommitteePeriodAtSlot(s.FinalizedHeader.Slot)
	updateSignaturePeriod := update.computeSyncCommitteePeriodAtSlot(update.GetSignatureSlot())
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
	updateAttestedPeriod := update.computeSyncCommitteePeriodAtSlot(update.GetAttestedHeader().Slot)
	updateHasNextSyncCommittee := !s.isNextSyncCommitteeKnown() && (update.isSyncCommiteeUpdate() && updateAttestedPeriod == storePeriod)
	if !(update.GetAttestedHeader().Slot > s.FinalizedHeader.Slot || updateHasNextSyncCommittee) {
		return errors.New("update is not relevant")
	}

	// Verify that the finality branch, if present, confirms finalized header to match the finalized checkpoint root
	// saved in the state of attested header. Note that the genesis finalized checkpoint root is represented as a zero
	// hash.
	if !update.isFinalityUpdate() {
		if update.GetFinalizedHeader() != nil {
			return errors.New("finality branch is present but update is not finality")
		}
	} else {
		var finalizedRoot [32]byte
		if update.GetFinalizedHeader().Slot == s.BeaconChainConfig.GenesisSlot {
			if update.GetFinalizedHeader().String() != (&ethpbv1.BeaconBlockHeader{}).String() {
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
	if !update.isSyncCommiteeUpdate() {
		if update.GetNextSyncCommittee() != nil {
			return errors.New("sync committee branch is present but update is not sync committee")
		}
	} else {
		if updateAttestedPeriod == storePeriod && s.isNextSyncCommitteeKnown() {
			if !update.GetNextSyncCommittee().Equals(s.NextSyncCommittee) {
				return errors.New("next sync committee is not known")
			}
		}
		root, err := hashTreeRoot(update.GetNextSyncCommittee())
		if err != nil {
			return err
		}
		if !trie.VerifyMerkleProofWithDepth(
			update.GetAttestedHeader().StateRoot,
			root[:],
			getSubtreeIndex(ethpbv2.NextSyncCommitteeIndex),
			update.GetNextSyncCommitteeBranch(),
			uint64(ethpbv2.FloorLog2(ethpbv2.NextSyncCommitteeIndex))) {
			return errors.New("sync committee branch is invalid")
		}
	}

	var syncCommittee *ethpbv2.SyncCommittee
	// Verify sync committee aggregate signature
	if updateSignaturePeriod == storePeriod {
		syncCommittee = s.CurrentSyncCommittee
	} else {
		syncCommittee = s.NextSyncCommittee
	}
	var participantPubkeys []common.PublicKey
	for i := uint64(0); i < syncAggregate.SyncCommitteeBits.Len(); i++ {
		bit := syncAggregate.SyncCommitteeBits.BitAt(i)
		if bit {
			publicKey, err := blst.PublicKeyFromBytes(syncCommittee.Pubkeys[i])
			if err != nil {
				return err
			}
			participantPubkeys = append(participantPubkeys, publicKey)
		}
	}
	forkVersion := s.computeForkVersion(update.computeEpochAtSlot(update.GetSignatureSlot()))
	domain, err := signing.ComputeDomain(s.BeaconChainConfig.DomainSyncCommittee, forkVersion, genesisValidatorsRoot)
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

func (s *Store) applyUpdate(update *Update) error {
	storePeriod := update.computeSyncCommitteePeriodAtSlot(s.FinalizedHeader.Slot)
	updateFinalizedPeriod := update.computeSyncCommitteePeriodAtSlot(update.GetFinalizedHeader().Slot)
	if !s.isNextSyncCommitteeKnown() {
		if updateFinalizedPeriod != storePeriod {
			return errors.New("update finalized period does not match store period")
		}
		s.NextSyncCommittee = update.GetNextSyncCommittee()
	} else if updateFinalizedPeriod == storePeriod+1 {
		s.CurrentSyncCommittee = s.NextSyncCommittee
		s.NextSyncCommittee = update.GetNextSyncCommittee()
		s.PreviousMaxActiveParticipants = s.CurrentMaxActiveParticipants
		s.CurrentMaxActiveParticipants = 0
	}
	if update.GetFinalizedHeader().Slot > s.FinalizedHeader.Slot {
		s.FinalizedHeader = update.GetFinalizedHeader()
		if s.FinalizedHeader.Slot > s.OptimisticHeader.Slot {
			s.OptimisticHeader = s.FinalizedHeader
		}
	}
	return nil
}

func (s *Store) ProcessForceUpdate(currentSlot types.Slot) error {
	if currentSlot > s.FinalizedHeader.Slot+s.BeaconChainConfig.SlotsPerEpoch+types.Slot(s.BeaconChainConfig.
		EpochsPerSyncCommitteePeriod) && s.BestValidUpdate != nil {
		// Forced best update when the update timeout has elapsed.
		// Because the apply logic waits for `finalized_header.slot` to indicate sync committee finality,
		// the `attested_header` may be treated as `finalized_header` in extended periods of non-finality
		// to guarantee progression into later sync committee periods according to `is_better_update`.
		if s.BestValidUpdate.GetFinalizedHeader().Slot <= s.FinalizedHeader.Slot {
			s.BestValidUpdate.SetFinalizedHeader(s.BestValidUpdate.GetAttestedHeader())
		}
		if err := s.applyUpdate(s.BestValidUpdate); err != nil {
			return err
		}
		s.BestValidUpdate = nil
	}
	return nil
}

func (s *Store) ProcessUpdate(update *Update,
	currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	if err := s.validateUpdate(update, currentSlot, genesisValidatorsRoot); err != nil {
		return err
	}
	syncCommiteeBits := update.GetSyncAggregate().SyncCommitteeBits

	// Update the best update in case we have to force-update to it if the timeout elapses
	if s.BestValidUpdate == nil || update.isBetterUpdate(s.BestValidUpdate) {
		s.BestValidUpdate = update
	}

	// Track the maximum number of active participants in the committee signature
	s.CurrentMaxActiveParticipants = max(s.CurrentMaxActiveParticipants, syncCommiteeBits.Count())

	// Update the optimistic header
	if syncCommiteeBits.Count() > s.getSafetyThreshold() && update.GetAttestedHeader().Slot > s.OptimisticHeader.Slot {
		s.OptimisticHeader = update.GetAttestedHeader()
	}

	// Update finalized header
	updateHasFinalizedNextSyncCommittee := !s.isNextSyncCommitteeKnown() && update.isSyncCommiteeUpdate() &&
		update.isFinalityUpdate() && update.computeSyncCommitteePeriodAtSlot(update.GetFinalizedHeader().
		Slot) == update.computeSyncCommitteePeriodAtSlot(update.GetAttestedHeader().Slot)
	if syncCommiteeBits.Count()*3 >= syncCommiteeBits.Len()*2 &&
		(update.GetFinalizedHeader().Slot > s.FinalizedHeader.Slot || updateHasFinalizedNextSyncCommittee) {
		// Normal update throught 2/3 threshold
		if err := s.applyUpdate(update); err != nil {
			return err
		}
		s.BestValidUpdate = nil
	}
	return nil
}

func (s *Store) ProcessFinalityUpdate(finalityUpdate *ethpbv2.FinalityUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	return s.ProcessUpdate(&Update{s.BeaconChainConfig, finalityUpdate}, currentSlot, genesisValidatorsRoot)
}

func (s *Store) ProcessOptimisticUpdate(optimisticUpdate *ethpbv2.OptimisticUpdate,
	currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	return s.ProcessUpdate(&Update{s.BeaconChainConfig, optimisticUpdate}, currentSlot, genesisValidatorsRoot)
}
