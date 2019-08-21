package helpers

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var activeIndicesCache = cache.NewActiveIndicesCache()
var activeCountCache = cache.NewActiveCountCache()

// IsActiveValidator returns the boolean value on whether the validator
// is active or not.
//
// Spec pseudocode definition:
//  def is_active_validator(validator: Validator, epoch: Epoch) -> bool:
//    """
//    Check if ``validator`` is active.
//    """
//    return validator.activation_epoch <= epoch < validator.exit_epoch
func IsActiveValidator(validator *ethpb.Validator, epoch uint64) bool {
	return validator.ActivationEpoch <= epoch &&
		epoch < validator.ExitEpoch
}

// IsSlashableValidator returns the boolean value on whether the validator
// is slashable or not.
//
// Spec pseudocode definition:
//  def is_slashable_validator(validator: Validator, epoch: Epoch) -> bool:
//    """
//    Check if ``validator`` is slashable.
//    """
//    return (
//        validator.activation_epoch <= epoch < validator.withdrawable_epoch and
//        validator.slashed is False
// 		)
func IsSlashableValidator(validator *ethpb.Validator, epoch uint64) bool {
	active := validator.ActivationEpoch <= epoch
	beforeWithdrawable := epoch < validator.WithdrawableEpoch
	return beforeWithdrawable && active && !validator.Slashed
}

// ActiveValidatorIndices filters out active validators based on validator status
// and returns their indices in a list.
//
// WARNING: This method allocates a new copy of the validator index set and is
// considered to be very memory expensive. Avoid using this unless you really
// need the active validator indices for some specific reason.
//
// Spec pseudocode definition:
//  def get_active_validator_indices(state: BeaconState, epoch: Epoch) -> Sequence[ValidatorIndex]:
//    """
//    Return the sequence of active validator indices at ``epoch``.
//    """
//    return [ValidatorIndex(i) for i, v in enumerate(state.validators) if is_active_validator(v, epoch)]
func ActiveValidatorIndices(state *pb.BeaconState, epoch uint64) ([]uint64, error) {
	indices, err := activeIndicesCache.ActiveIndicesInEpoch(epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve active indices from cache")
	}
	if indices != nil {
		return indices, nil
	}

	for i, v := range state.Validators {
		if IsActiveValidator(v, epoch) {
			indices = append(indices, uint64(i))
		}
	}

	if err := activeIndicesCache.AddActiveIndicesList(&cache.ActiveIndicesByEpoch{
		Epoch:         epoch,
		ActiveIndices: indices,
	}); err != nil {
		return nil, errors.Wrap(err, "could not save active indices for cache")
	}

	return indices, nil
}

// ActiveValidatorCount returns the number of active validators in the state
// at the given epoch.
func ActiveValidatorCount(state *pb.BeaconState, epoch uint64) (uint64, error) {
	count, err := activeCountCache.ActiveCountInEpoch(epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not retrieve active count from cache")
	}
	if count != params.BeaconConfig().FarFutureEpoch {
		return count, nil
	}

	count = 0
	for _, v := range state.Validators {
		if IsActiveValidator(v, epoch) {
			count++
		}
	}

	if err := activeCountCache.AddActiveCount(&cache.ActiveCountByEpoch{
		Epoch:       epoch,
		ActiveCount: count,
	}); err != nil {
		return 0, errors.Wrap(err, "could not save active count for cache")
	}

	return count, nil
}

// DelayedActivationExitEpoch takes in epoch number and returns when
// the validator is eligible for activation and exit.
//
// Spec pseudocode definition:
//  def compute_activation_exit_epoch(epoch: Epoch) -> Epoch:
//    """
//    Return the epoch during which validator activations and exits initiated in ``epoch`` take effect.
//    """
//    return Epoch(epoch + 1 + ACTIVATION_EXIT_DELAY)
func DelayedActivationExitEpoch(epoch uint64) uint64 {
	return epoch + 1 + params.BeaconConfig().ActivationExitDelay
}

// ValidatorChurnLimit returns the number of validators that are allowed to
// enter and exit validator pool for an epoch.
//
// Spec pseudocode definition:
//   def get_validator_churn_limit(state: BeaconState) -> uint64:
//    """
//    Return the validator churn limit for the current epoch.
//    """
//    active_validator_indices = get_active_validator_indices(state, get_current_epoch(state))
//    return max(MIN_PER_EPOCH_CHURN_LIMIT, len(active_validator_indices) // CHURN_LIMIT_QUOTIENT)
func ValidatorChurnLimit(state *pb.BeaconState) (uint64, error) {
	validatorCount, err := ActiveValidatorCount(state, CurrentEpoch(state))
	if err != nil {
		return 0, errors.Wrap(err, "could not get validator count")
	}
	churnLimit := validatorCount / params.BeaconConfig().ChurnLimitQuotient
	if churnLimit < params.BeaconConfig().MinPerEpochChurnLimit {
		churnLimit = params.BeaconConfig().MinPerEpochChurnLimit
	}
	return churnLimit, nil
}

// BeaconProposerIndex returns proposer index of a current slot.
//
// Spec pseudocode definition:
//  def get_beacon_proposer_index(state: BeaconState) -> ValidatorIndex:
//    """
//    Return the beacon proposer index at the current slot.
//    """
//    epoch = get_current_epoch(state)
//    committees_per_slot = get_committee_count(state, epoch) // SLOTS_PER_EPOCH
//    offset = committees_per_slot * (state.slot % SLOTS_PER_EPOCH)
//    shard = Shard((get_start_shard(state, epoch) + offset) % SHARD_COUNT)
//    first_committee = get_crosslink_committee(state, epoch, shard)
//    MAX_RANDOM_BYTE = 2**8 - 1
//    seed = get_seed(state, epoch)
//    i = 0
//    while True:
//        candidate_index = first_committee[(epoch + i) % len(first_committee)]
//        random_byte = hash(seed + int_to_bytes(i // 32, length=8))[i % 32]
//        effective_balance = state.validators[candidate_index].effective_balance
//        if effective_balance * MAX_RANDOM_BYTE >= MAX_EFFECTIVE_BALANCE * random_byte:
//            return ValidatorIndex(candidate_index)
//        i += 1
func BeaconProposerIndex(state *pb.BeaconState) (uint64, error) {
	// Calculate the offset for slot and shard
	e := CurrentEpoch(state)
	committeeCount, err := CommitteeCount(state, e)
	if err != nil {
		return 0, err
	}
	committesPerSlot := committeeCount / params.BeaconConfig().SlotsPerEpoch
	offSet := committesPerSlot * (state.Slot % params.BeaconConfig().SlotsPerEpoch)

	// Calculate which shards get assigned given the epoch start shard
	// and the offset
	startShard, err := StartShard(state, e)
	if err != nil {
		return 0, errors.Wrap(err, "could not get start shard")
	}
	shard := (startShard + offSet) % params.BeaconConfig().ShardCount

	// Use the first committee of the given slot and shard
	// to select proposer
	firstCommittee, err := CrosslinkCommittee(state, e, shard)
	if err != nil {
		return 0, errors.Wrap(err, "could not get first committee")
	}
	if len(firstCommittee) == 0 {
		return 0, fmt.Errorf("empty first committee at slot %d", state.Slot)
	}

	// Use the generated seed to select proposer from the first committee
	maxRandomByte := uint64(1<<8 - 1)
	seed, err := Seed(state, e)
	if err != nil {
		return 0, errors.Wrap(err, "could not generate seed")
	}

	// Looping through the committee to select proposer that has enough
	// effective balance.
	for i := uint64(0); ; i++ {
		candidateIndex := firstCommittee[(e+i)%uint64(len(firstCommittee))]
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashutil.Hash(b)[i%32]
		effectiveBal := state.Validators[candidateIndex].EffectiveBalance
		if effectiveBal*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}

// Domain returns the domain version for BLS private key to sign and verify.
//
// Spec pseudocode definition:
//  def get_domain(state: BeaconState,
//               domain_type: int,
//               message_epoch: Epoch=None) -> int:
//    """
//    Return the signature domain (fork version concatenated with domain type) of a message.
//    """
//    epoch = get_current_epoch(state) if message_epoch is None else message_epoch
//    fork_version = state.fork.previous_version if epoch < state.fork.epoch else state.fork.current_version
//    return bls_domain(domain_type, fork_version)
func Domain(state *pb.BeaconState, epoch uint64, domainType []byte) uint64 {
	var forkVersion []byte
	if epoch < state.Fork.Epoch {
		forkVersion = state.Fork.PreviousVersion
	} else {
		forkVersion = state.Fork.CurrentVersion
	}
	return bls.Domain(domainType, forkVersion)
}
