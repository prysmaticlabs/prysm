// Package helpers contains helper functions outlined in the eth2 beacon chain spec, such as
// computing committees, randao, rewards/penalties, and more.
package helpers

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

var committeeCache = cache.NewCommitteesCache()

// SlotCommitteeCount returns the number of crosslink committees of a slot. The
// active validator count is provided as an argument rather than a direct implementation
// from the spec definition. Having the active validator count as an argument allows for
// cheaper computation, instead of retrieving head state, one can retrieve the validator
// count.
//
//
// Spec pseudocode definition:
//   def get_committee_count_at_slot(state: BeaconState, slot: Slot) -> uint64:
//    """
//    Return the number of committees at ``slot``.
//    """
//    epoch = compute_epoch_at_slot(slot)
//    return max(1, min(
//        MAX_COMMITTEES_PER_SLOT,
//        len(get_active_validator_indices(state, epoch)) // SLOTS_PER_EPOCH // TARGET_COMMITTEE_SIZE,
//    ))
func SlotCommitteeCount(activeValidatorCount uint64) uint64 {
	var committeePerSlot = activeValidatorCount / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize

	if committeePerSlot > params.BeaconConfig().MaxCommitteesPerSlot {
		return params.BeaconConfig().MaxCommitteesPerSlot
	}
	if committeePerSlot == 0 {
		return 1
	}

	return committeePerSlot
}

// BeaconCommitteeFromState returns the crosslink committee of a given slot and committee index. This
// is a spec implementation where state is used as an argument. In case of state retrieval
// becomes expensive, consider using BeaconCommittee below.
//
// Spec pseudocode definition:
//   def get_beacon_committee(state: BeaconState, slot: Slot, index: CommitteeIndex) -> Sequence[ValidatorIndex]:
//    """
//    Return the beacon committee at ``slot`` for ``index``.
//    """
//    epoch = compute_epoch_at_slot(slot)
//    committees_per_slot = get_committee_count_per_slot(state, epoch)
//    return compute_committee(
//        indices=get_active_validator_indices(state, epoch),
//        seed=get_seed(state, epoch, DOMAIN_BEACON_ATTESTER),
//        index=(slot % SLOTS_PER_EPOCH) * committees_per_slot + index,
//        count=committees_per_slot * SLOTS_PER_EPOCH,
//    )
func BeaconCommitteeFromState(state *stateTrie.BeaconState, slot uint64, committeeIndex uint64) ([]uint64, error) {
	epoch := SlotToEpoch(slot)
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seed")
	}

	indices, err := committeeCache.Committee(slot, seed, committeeIndex)
	if err != nil {
		return nil, errors.Wrap(err, "could not interface with committee cache")
	}
	if indices != nil {
		return indices, nil
	}

	activeIndices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active indices")
	}

	return BeaconCommittee(activeIndices, seed, slot, committeeIndex)
}

// BeaconCommittee returns the crosslink committee of a given slot and committee index. The
// validator indices and seed are provided as an argument rather than a direct implementation
// from the spec definition. Having them as an argument allows for cheaper computation run time.
func BeaconCommittee(validatorIndices []uint64, seed [32]byte, slot uint64, committeeIndex uint64) ([]uint64, error) {
	indices, err := committeeCache.Committee(slot, seed, committeeIndex)
	if err != nil {
		return nil, errors.Wrap(err, "could not interface with committee cache")
	}
	if indices != nil {
		return indices, nil
	}

	committeesPerSlot := SlotCommitteeCount(uint64(len(validatorIndices)))

	epochOffset := committeeIndex + (slot%params.BeaconConfig().SlotsPerEpoch)*committeesPerSlot
	count := committeesPerSlot * params.BeaconConfig().SlotsPerEpoch

	return ComputeCommittee(validatorIndices, seed, epochOffset, count)
}

// ComputeCommittee returns the requested shuffled committee out of the total committees using
// validator indices and seed.
//
// Spec pseudocode definition:
//  def compute_committee(indices: Sequence[ValidatorIndex],
//                      seed: Hash,
//                      index: uint64,
//                      count: uint64) -> Sequence[ValidatorIndex]:
//    """
//    Return the committee corresponding to ``indices``, ``seed``, ``index``, and committee ``count``.
//    """
//    start = (len(indices) * index) // count
//    end = (len(indices) * (index + 1)) // count
//    return [indices[compute_shuffled_index(ValidatorIndex(i), len(indices), seed)] for i in range(start, end)
func ComputeCommittee(
	indices []uint64,
	seed [32]byte,
	index uint64,
	count uint64,
) ([]uint64, error) {
	validatorCount := uint64(len(indices))
	start := sliceutil.SplitOffset(validatorCount, count, index)
	end := sliceutil.SplitOffset(validatorCount, count, index+1)

	// Save the shuffled indices in cache, this is only needed once per epoch or once per new committee index.
	shuffledIndices := make([]uint64, len(indices))
	copy(shuffledIndices, indices)
	// UnshuffleList is used here as it is an optimized implementation created
	// for fast computation of committees.
	// Reference implementation: https://github.com/protolambda/eth2-shuffle
	shuffledList, err := UnshuffleList(shuffledIndices, seed)
	return shuffledList[start:end], err
}

// AttestingIndices returns the attesting participants indices from the attestation data. The
// committee is provided as an argument rather than a direct implementation from the spec definition.
// Having the committee as an argument allows for re-use of beacon committees when possible.
//
// Spec pseudocode definition:
//   def get_attesting_indices(state: BeaconState,
//                          data: AttestationData,
//                          bits: Bitlist[MAX_VALIDATORS_PER_COMMITTEE]) -> Set[ValidatorIndex]:
//    """
//    Return the set of attesting indices corresponding to ``data`` and ``bits``.
//    """
//    committee = get_beacon_committee(state, data.slot, data.index)
//    return set(index for i, index in enumerate(committee) if bits[i])
func AttestingIndices(bf bitfield.Bitfield, committee []uint64) ([]uint64, error) {
	indices := make([]uint64, 0, len(committee))
	indicesSet := make(map[uint64]bool, len(committee))
	for i, idx := range committee {
		if !indicesSet[idx] {
			if bf.BitAt(uint64(i)) {
				indices = append(indices, idx)
			}
		}
		indicesSet[idx] = true
	}
	return indices, nil
}

// CommitteeAssignmentContainer represents a committee, index, and attester slot for a given epoch.
type CommitteeAssignmentContainer struct {
	Committee      []uint64
	AttesterSlot   uint64
	CommitteeIndex uint64
}

// CommitteeAssignments is a map of validator indices pointing to the appropriate committee
// assignment for the given epoch.
//
// 1. Determine the proposer validator index for each slot.
// 2. Compute all committees.
// 3. Determine the attesting slot for each committee.
// 4. Construct a map of validator indices pointing to the respective committees.
func CommitteeAssignments(
	state *stateTrie.BeaconState,
	epoch uint64,
) (map[uint64]*CommitteeAssignmentContainer, map[uint64][]uint64, error) {
	nextEpoch := NextEpoch(state)
	if epoch > nextEpoch {
		return nil, nil, fmt.Errorf(
			"epoch %d can't be greater than next epoch %d",
			epoch,
			nextEpoch,
		)
	}

	// We determine the slots in which proposers are supposed to act.
	// Some validators may need to propose multiple times per epoch, so
	// we use a map of proposer idx -> []slot to keep track of this possibility.
	startSlot := StartSlot(epoch)
	proposerIndexToSlots := make(map[uint64][]uint64)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		// Skip proposer assignment for genesis slot.
		if slot == 0 {
			continue
		}
		if err := state.SetSlot(slot); err != nil {
			return nil, nil, err
		}
		i, err := BeaconProposerIndex(state)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "could not check proposer at slot %d", state.Slot())
		}
		proposerIndexToSlots[i] = append(proposerIndexToSlots[i], slot)
	}

	activeValidatorIndices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, nil, err
	}
	// Each slot in an epoch has a different set of committees. This value is derived from the
	// active validator set, which does not change.
	numCommitteesPerSlot := SlotCommitteeCount(uint64(len(activeValidatorIndices)))
	validatorIndexToCommittee := make(map[uint64]*CommitteeAssignmentContainer)

	// Compute all committees for all slots.
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		// Compute committees.
		for j := uint64(0); j < numCommitteesPerSlot; j++ {
			slot := startSlot + i
			committee, err := BeaconCommitteeFromState(state, slot, j /*committee index*/)
			if err != nil {
				return nil, nil, err
			}

			cac := &CommitteeAssignmentContainer{
				Committee:      committee,
				CommitteeIndex: j,
				AttesterSlot:   slot,
			}
			for _, vID := range committee {
				validatorIndexToCommittee[vID] = cac
			}
		}
	}

	return validatorIndexToCommittee, proposerIndexToSlots, nil
}

// VerifyBitfieldLength verifies that a bitfield length matches the given committee size.
func VerifyBitfieldLength(bf bitfield.Bitfield, committeeSize uint64) error {
	if bf.Len() != committeeSize {
		return fmt.Errorf(
			"wanted participants bitfield length %d, got: %d",
			committeeSize,
			bf.Len())
	}
	return nil
}

// VerifyAttestationBitfieldLengths verifies that an attestations aggregation bitfields is
// a valid length matching the size of the committee.
func VerifyAttestationBitfieldLengths(state *stateTrie.BeaconState, att *ethpb.Attestation) error {
	committee, err := BeaconCommitteeFromState(state, att.Data.Slot, att.Data.CommitteeIndex)
	if err != nil {
		return errors.Wrap(err, "could not retrieve beacon committees")
	}

	if committee == nil {
		return errors.New("no committee exist for this attestation")
	}

	if err := VerifyBitfieldLength(att.AggregationBits, uint64(len(committee))); err != nil {
		return errors.Wrap(err, "failed to verify aggregation bitfield")
	}
	return nil
}

// ShuffledIndices uses input beacon state and returns the shuffled indices of the input epoch,
// the shuffled indices then can be used to break up into committees.
func ShuffledIndices(state *stateTrie.BeaconState, epoch uint64) ([]uint64, error) {
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get seed for epoch %d", epoch)
	}

	indices := make([]uint64, 0, state.NumValidators())
	if err := state.ReadFromEveryValidator(func(idx int, val *stateTrie.ReadOnlyValidator) error {
		if IsActiveValidatorUsingTrie(val, epoch) {
			indices = append(indices, uint64(idx))
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// UnshuffleList is used as an optimized implementation for raw speed.
	return UnshuffleList(indices, seed)
}

// UpdateCommitteeCache gets called at the beginning of every epoch to cache the committee shuffled indices
// list with committee index and epoch number. It caches the shuffled indices for current epoch and next epoch.
func UpdateCommitteeCache(state *stateTrie.BeaconState, epoch uint64) error {
	for _, e := range []uint64{epoch, epoch + 1} {
		seed, err := Seed(state, e, params.BeaconConfig().DomainBeaconAttester)
		if err != nil {
			return err
		}
		if _, exists, err := committeeCache.CommitteeCache.GetByKey(string(seed[:])); err == nil && exists {
			return nil
		}

		shuffledIndices, err := ShuffledIndices(state, e)
		if err != nil {
			return err
		}

		count := SlotCommitteeCount(uint64(len(shuffledIndices)))

		// Store the sorted indices as well as shuffled indices. In current spec,
		// sorted indices is required to retrieve proposer index. This is also
		// used for failing verify signature fallback.
		sortedIndices := make([]uint64, len(shuffledIndices))
		copy(sortedIndices, shuffledIndices)
		sort.Slice(sortedIndices, func(i, j int) bool {
			return sortedIndices[i] < sortedIndices[j]
		})

		if err := committeeCache.AddCommitteeShuffledList(&cache.Committees{
			ShuffledIndices: shuffledIndices,
			CommitteeCount:  count * params.BeaconConfig().SlotsPerEpoch,
			Seed:            seed,
			SortedIndices:   sortedIndices,
		}); err != nil {
			return err
		}
	}

	return nil
}

// UpdateProposerIndicesInCache updates proposer indices entry of the committee cache.
func UpdateProposerIndicesInCache(state *stateTrie.BeaconState, epoch uint64) error {
	indices, err := ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil
	}
	proposerIndices, err := precomputeProposerIndices(state, indices)
	if err != nil {
		return err
	}
	// The committee cache uses attester domain seed as key.
	seed, err := Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return err
	}
	if err := committeeCache.AddProposerIndicesList(seed, proposerIndices); err != nil {
		return err
	}

	return nil
}

// ClearCache clears the committee cache
func ClearCache() {
	committeeCache = cache.NewCommitteesCache()
}

// This computes proposer indices of the current epoch and returns a list of proposer indices,
// the index of the list represents the slot number.
func precomputeProposerIndices(state *stateTrie.BeaconState, activeIndices []uint64) ([]uint64, error) {
	hashFunc := hashutil.CustomSHA256Hasher()
	proposerIndices := make([]uint64, params.BeaconConfig().SlotsPerEpoch)

	e := CurrentEpoch(state)
	seed, err := Seed(state, e, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate seed")
	}
	slot := StartSlot(e)
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		seedWithSlot := append(seed[:], bytesutil.Bytes8(slot+i)...)
		seedWithSlotHash := hashFunc(seedWithSlot)
		index, err := ComputeProposerIndex(state, activeIndices, seedWithSlotHash)
		if err != nil {
			return nil, err
		}
		proposerIndices[i] = index
	}

	return proposerIndices, nil
}
