// Package lightclient implements the light client for the Ethereum 2.0 Beacon Chain.
// It is based on the Altair light client spec at this revision:
// https://github.com/ethereum/consensus-specs/tree/208da34ac4e75337baf79adebf036ab595e39f15/specs/altair/light-client
package lightclient

import (
	"errors"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	types "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/container/trie"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/blst"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls/common"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
)

const (
	currentSyncCommitteeIndex = uint64(54)
)

// Store implements LightClientStore from the spec.
type Store struct {
	Config *Config `json:"config,omitempty"`
	// FinalizedHeader is a header that is finalized
	FinalizedHeader *ethpbv1.BeaconBlockHeader `json:"finalized_header,omitempty"`
	// CurrentSyncCommittee is the sync committees corresponding to the finalized header
	CurrentSyncCommittee *ethpbv2.SyncCommittee `json:"current_sync_committeeu,omitempty"`
	// NextSyncCommittee is the next sync committees corresponding to the finalized header
	NextSyncCommittee *ethpbv2.SyncCommittee `json:"next_sync_committee,omitempty"`
	// BestValidUpdate is the best available header to switch finalized head to if we see nothing else
	BestValidUpdate *ethpbv2.LightClientUpdate `json:"best_valid_update,omitempty"`
	// OptimisticHeader is the most recent available reasonably-safe header
	OptimisticHeader *ethpbv1.BeaconBlockHeader `json:"optimistic_header,omitempty"`
	// PreviousMaxActiveParticipants is the previous max number of active participants in a sync committee (used to
	// calculate safety threshold)
	PreviousMaxActiveParticipants uint64 `json:"previous_max_active_participants,omitempty"`
	// CurrentMaxActiveParticipants is the max number of active participants in a sync committee (used to calculate
	// safety threshold)
	CurrentMaxActiveParticipants uint64 `json:"current_max_active_participants,omitempty"`
}

// getSubtreeIndex implements get_subtree_index from the spec.
func getSubtreeIndex(index uint64) uint64 {
	return index % (uint64(1) << ethpbv2.FloorLog2(index-1))
}

// NewStore implements initialize_light_client_store from the spec.
func NewStore(config *Config, trustedBlockRoot [32]byte,
	bootstrap *ethpbv2.LightClientBootstrap) (*Store, error) {
	if config == nil {
		return nil, errors.New("light client config cannot be nil")
	}
	if bootstrap.Header == nil || bootstrap.CurrentSyncCommittee == nil {
		return nil, errors.New("malformed bootstrap")
	}
	bootstrapRoot, err := bootstrap.Header.HashTreeRoot()
	if err != nil {
		panic(err)
	}
	if trustedBlockRoot != bootstrapRoot {
		return nil, errors.New("trusted block root does not match bootstrap header")
	}
	root, err := bootstrap.CurrentSyncCommittee.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	if !trie.VerifyMerkleProof(
		bootstrap.Header.StateRoot,
		root[:],
		getSubtreeIndex(currentSyncCommitteeIndex),
		bootstrap.CurrentSyncCommitteeBranch) {
		return nil, errors.New("current sync committee merkle proof is invalid")
	}
	return &Store{
		Config:               config,
		FinalizedHeader:      bootstrap.Header,
		CurrentSyncCommittee: bootstrap.CurrentSyncCommittee,
		NextSyncCommittee:    nil,
		OptimisticHeader:     bootstrap.Header,
	}, nil
}

func (s *Store) Clone() *Store {
	return &Store{
		Config:                        s.Config,
		FinalizedHeader:               s.FinalizedHeader,
		CurrentSyncCommittee:          s.CurrentSyncCommittee,
		NextSyncCommittee:             s.NextSyncCommittee,
		BestValidUpdate:               s.BestValidUpdate,
		OptimisticHeader:              s.OptimisticHeader,
		PreviousMaxActiveParticipants: s.PreviousMaxActiveParticipants,
		CurrentMaxActiveParticipants:  s.CurrentMaxActiveParticipants,
	}
}

// isNextSyncCommitteeKnown implements is_next_sync_committee_known from the spec.
func (s *Store) isNextSyncCommitteeKnown() bool {
	return s.NextSyncCommittee != nil
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// getSafetyThreshold implements get_safety_threshold from the spec.
func (s *Store) getSafetyThreshold() uint64 {
	return max(s.PreviousMaxActiveParticipants, s.CurrentMaxActiveParticipants) / 2
}

// computeForkVersion implements compute_fork_version from the spec.
func (s *Store) computeForkVersion(epoch types.Epoch) []byte {
	if epoch >= s.Config.DenebForkEpoch {
		return s.Config.DenebForkVersion
	}
	if epoch >= s.Config.CapellaForkEpoch {
		return s.Config.CapellaForkVersion
	}
	if epoch >= s.Config.BellatrixForkEpoch {
		return s.Config.BellatrixForkVersion
	}
	if epoch >= s.Config.AltairForkEpoch {
		return s.Config.AltairForkVersion
	}
	return s.Config.GenesisForkVersion
}

// validateWrappedUpdate implements validate_light_client_update from the spec.
func (s *Store) validateWrappedUpdate(update *update, currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	// Verify sync committee has sufficient participants
	syncAggregate := update.SyncAggregate
	if syncAggregate == nil || syncAggregate.SyncCommitteeBits == nil {
		return errors.New("sync aggregate in update is invalid")
	}
	if syncAggregate.SyncCommitteeBits.Count() < s.Config.MinSyncCommitteeParticipants {
		return errors.New("sync committee does not have sufficient participants")
	}

	if update.AttestedHeader == nil {
		return errors.New("attested header in update is not set")
	}
	// Verify update does not skip a sync committee period
	if !(currentSlot >= update.SignatureSlot &&
		update.SignatureSlot > update.AttestedHeader.Slot &&
		(update.FinalizedHeader == nil || update.AttestedHeader.Slot >= update.FinalizedHeader.Slot)) {
		return errors.New("update skips a sync committee period")
	}
	storePeriod := computeSyncCommitteePeriodAtSlot(s.Config, s.FinalizedHeader.Slot)
	updateSignaturePeriod := computeSyncCommitteePeriodAtSlot(s.Config, update.SignatureSlot)
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
	updateAttestedPeriod := computeSyncCommitteePeriodAtSlot(s.Config, update.AttestedHeader.Slot)
	updateHasNextSyncCommittee := !s.isNextSyncCommitteeKnown() && (update.IsSyncCommiteeUpdate() && updateAttestedPeriod == storePeriod)
	if !(update.AttestedHeader.Slot > s.FinalizedHeader.Slot || updateHasNextSyncCommittee) {
		return errors.New("update is not relevant")
	}

	// Verify that the finality branch, if present, confirms finalized header to match the finalized checkpoint root
	// saved in the state of attested header. Note that the genesis finalized checkpoint root is represented as a zero
	// hash.
	if !update.IsFinalityUpdate() {
		if update.FinalizedHeader != nil {
			return errors.New("finality branch is present but update is not finality")
		}
	} else {
		var finalizedRoot [32]byte
		if update.FinalizedHeader.Slot == s.Config.GenesisSlot {
			if update.FinalizedHeader.String() != (&ethpbv1.BeaconBlockHeader{}).String() {
				return errors.New("genesis finalized checkpoint root is not represented as a zero hash")
			}
			finalizedRoot = [32]byte{}
		} else {
			var err error
			if finalizedRoot, err = update.FinalizedHeader.HashTreeRoot(); err != nil {
				return err
			}
		}
		if !trie.VerifyMerkleProof(
			update.AttestedHeader.StateRoot,
			finalizedRoot[:],
			getSubtreeIndex(ethpbv2.FinalizedRootIndex),
			update.FinalityBranch) {
			return errors.New("finality branch is invalid")
		}
	}

	// Verify that the next sync committee, if present, actually is the next sync committee saved in the state of the
	// attested header
	if !update.IsSyncCommiteeUpdate() {
		if update.NextSyncCommittee != nil {
			return errors.New("sync committee branch is present but update is not sync committee")
		}
	} else {
		if updateAttestedPeriod == storePeriod && s.isNextSyncCommitteeKnown() {
			if !update.NextSyncCommittee.Equals(s.NextSyncCommittee) {
				return errors.New("next sync committee is not known")
			}
		}
		root, err := update.NextSyncCommittee.HashTreeRoot()
		if err != nil {
			return err
		}
		if !trie.VerifyMerkleProof(
			update.AttestedHeader.StateRoot,
			root[:],
			getSubtreeIndex(ethpbv2.NextSyncCommitteeIndex),
			update.NextSyncCommitteeBranch) {
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
	forkVersion := s.computeForkVersion(computeEpochAtSlot(s.Config, update.SignatureSlot))
	domain, err := signing.ComputeDomain(s.Config.DomainSyncCommittee, forkVersion, genesisValidatorsRoot)
	if err != nil {
		return err
	}
	signingRoot, err := signing.ComputeSigningRoot(update.AttestedHeader, domain)
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

// applyUpdate implements apply_light_client_update from the spec.
func (s *Store) applyUpdate(update *ethpbv2.LightClientUpdate) error {
	storePeriod := computeSyncCommitteePeriodAtSlot(s.Config, s.FinalizedHeader.Slot)
	updateFinalizedPeriod := computeSyncCommitteePeriodAtSlot(s.Config, update.FinalizedHeader.Slot)
	if !s.isNextSyncCommitteeKnown() {
		if updateFinalizedPeriod != storePeriod {
			return errors.New("update finalized period does not match store period")
		}
		s.NextSyncCommittee = update.NextSyncCommittee
	} else if updateFinalizedPeriod == storePeriod+1 {
		s.CurrentSyncCommittee = s.NextSyncCommittee
		s.NextSyncCommittee = update.NextSyncCommittee
		s.PreviousMaxActiveParticipants = s.CurrentMaxActiveParticipants
		s.CurrentMaxActiveParticipants = 0
	}
	if update.FinalizedHeader.Slot > s.FinalizedHeader.Slot {
		s.FinalizedHeader = update.FinalizedHeader
		if s.FinalizedHeader.Slot > s.OptimisticHeader.Slot {
			s.OptimisticHeader = s.FinalizedHeader
		}
	}
	return nil
}

// ProcessForceUpdate implements process_light_client_store_force_update from the spec.
func (s *Store) ProcessForceUpdate(currentSlot types.Slot) error {
	if currentSlot > s.FinalizedHeader.Slot+s.Config.SlotsPerEpoch+types.Slot(s.Config.EpochsPerSyncCommitteePeriod) &&
		s.BestValidUpdate != nil {
		// Forced best update when the update timeout has elapsed.
		// Because the apply logic waits for `finalized_header.slot` to indicate sync committee finality,
		// the `attested_header` may be treated as `finalized_header` in extended periods of non-finality
		// to guarantee progression into later sync committee periods according to `is_better_update`.
		if s.BestValidUpdate.FinalizedHeader.Slot <= s.FinalizedHeader.Slot {
			s.BestValidUpdate.FinalizedHeader = s.BestValidUpdate.AttestedHeader
		}
		if err := s.applyUpdate(s.BestValidUpdate); err != nil {
			return err
		}
		s.BestValidUpdate = nil
	}
	return nil
}

// ValidateUpdate provides a wrapper around validateUpdate() for callers that want to separate validate and apply.
func (s *Store) ValidateUpdate(lightClientUpdate *ethpbv2.LightClientUpdate,
	currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	update := &update{
		LightClientUpdate: lightClientUpdate,
		config:            s.Config,
	}
	return s.validateWrappedUpdate(update, currentSlot, genesisValidatorsRoot)
}

func (s *Store) maybeValidateAndProcessUpdate(lightClientUpdate *ethpbv2.LightClientUpdate,
	currentSlot types.Slot, genesisValidatorsRoot []byte, validated bool) error {
	update := &update{
		LightClientUpdate: lightClientUpdate,
		config:            s.Config,
	}
	if !validated {
		if err := s.validateWrappedUpdate(update, currentSlot, genesisValidatorsRoot); err != nil {
			return err
		}
	}
	syncCommiteeBits := update.SyncAggregate.SyncCommitteeBits

	// Update the best update in case we have to force-update to it if the timeout elapses
	if s.BestValidUpdate == nil || update.isBetterUpdate(s.BestValidUpdate) {
		s.BestValidUpdate = update.LightClientUpdate
	}

	// Track the maximum number of active participants in the committee signature
	s.CurrentMaxActiveParticipants = max(s.CurrentMaxActiveParticipants, syncCommiteeBits.Count())

	// Update the optimistic header
	if syncCommiteeBits.Count() > s.getSafetyThreshold() && update.AttestedHeader.Slot > s.OptimisticHeader.Slot {
		s.OptimisticHeader = update.AttestedHeader
	}

	// Update finalized header
	updateHasFinalizedNextSyncCommittee := !s.isNextSyncCommitteeKnown() && update.IsSyncCommiteeUpdate() &&
		update.IsFinalityUpdate() && computeSyncCommitteePeriodAtSlot(s.Config, update.FinalizedHeader.
		Slot) == computeSyncCommitteePeriodAtSlot(s.Config, update.AttestedHeader.Slot)
	if syncCommiteeBits.Count()*3 >= syncCommiteeBits.Len()*2 &&
		((update.FinalizedHeader != nil && update.FinalizedHeader.Slot > s.FinalizedHeader.Slot) ||
			updateHasFinalizedNextSyncCommittee) {
		// Normal update through 2/3 threshold
		if err := s.applyUpdate(update.LightClientUpdate); err != nil {
			return err
		}
		s.BestValidUpdate = nil
	}
	return nil
}

// ProcessUpdate implements process_light_client_update from the spec.
func (s *Store) ProcessUpdate(lightClientUpdate *ethpbv2.LightClientUpdate,
	currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	return s.maybeValidateAndProcessUpdate(lightClientUpdate, currentSlot, genesisValidatorsRoot, false)
}

// ProcessValidatedUpdate processes a pre-validated update.
func (s *Store) ProcessValidatedUpdate(lightClientUpdate *ethpbv2.LightClientUpdate,
	currentSlot types.Slot, genesisValidatorsRoot []byte) error {
	return s.maybeValidateAndProcessUpdate(lightClientUpdate, currentSlot, genesisValidatorsRoot, true)
}

// ProcessFinalityUpdate implements process_light_client_finality_update from the spec.
func (s *Store) ProcessFinalityUpdate(update *ethpbv2.LightClientFinalityUpdate, currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	return s.ProcessUpdate(blockchain.NewLightClientUpdateFromFinalityUpdate(update), currentSlot,
		genesisValidatorsRoot)
}

// ProcessOptimisticUpdate implements process_light_client_optimistic_update from the spec.
func (s *Store) ProcessOptimisticUpdate(update *ethpbv2.LightClientOptimisticUpdate, currentSlot types.Slot,
	genesisValidatorsRoot []byte) error {
	return s.ProcessUpdate(blockchain.NewLightClientUpdateFromOptimisticUpdate(update), currentSlot,
		genesisValidatorsRoot)
}
