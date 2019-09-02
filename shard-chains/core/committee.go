package core

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// PeriodCommittee returns the period committee of a given period.
//
// Spec pseudocode definition:
//  def get_period_committee(state: BeaconState, shard: Shard, epoch: Epoch) -> Sequence[ValidatorIndex]:
//    active_validator_indices = get_active_validator_indices(state, epoch)
//    seed = get_seed(state, epoch)
//    return compute_committee(active_validator_indices, seed, shard, SHARD_COUNT)[:MAX_PERIOD_COMMITTEE_SIZE]
func PeriodCommittee(state *pb.BeaconState, shard uint64, epoch uint64) ([]uint64, error) {
	activeIndices, err := helpers.ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active indices")
	}
	seed, err := helpers.Seed(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seed")
	}
	committee, err := helpers.ComputeCommittee(activeIndices, seed, shard, params.BeaconConfig().ShardCount)
	if err != nil {
		return nil, errors.Wrap(err, "could not get committee")
	}

	return committee[:params.BeaconConfig().MaxPeriodCommitteeSize], nil
}

// ShardCommittee returns the shard committee of a given period.
//
// Spec pseudocode definition:
//  def get_shard_committee(state: BeaconState, shard: Shard, epoch: Epoch) -> Sequence[ValidatorIndex]:
//    older_committee = get_period_committee(state, shard, compute_shard_period_start_epoch(epoch, 2))
//    newer_committee = get_period_committee(state, shard, compute_shard_period_start_epoch(epoch, 1))
//    # Every epoch cycle out validators from the older committee and cycle in validators from the newer committee
//    older_subcommittee = [i for i in older_committee if i % EPOCHS_PER_SHARD_PERIOD > epoch % EPOCHS_PER_SHARD_PERIOD]
//    newer_subcommittee = [i for i in newer_committee if i % EPOCHS_PER_SHARD_PERIOD <= epoch % EPOCHS_PER_SHARD_PERIOD]
//    return older_subcommittee + newer_subcommittee
func ShardCommittee(state *pb.BeaconState, shard uint64, epoch uint64) ([]uint64, error) {
	olderEpoch := ShardPeriodStartEpoch(epoch, 2)
	newerEpoch := ShardPeriodStartEpoch(epoch, 1)
	olderCommittee, err := PeriodCommittee(state, shard, olderEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get older committee")
	}
	newerCommittee, err := PeriodCommittee(state, shard, newerEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get newer committee")
	}

	// For every epoch, cycle out validators from older committee and cycle in validators from newer committee.
	olderSubCommittee := make([]uint64, 0, len(olderCommittee))
	newerSubCommittee := make([]uint64, 0, len(newerCommittee))

	for _, index := range olderCommittee {
		if index%params.BeaconConfig().EpochsPerShardPeriod > epoch%params.BeaconConfig().EpochsPerShardPeriod {
			olderSubCommittee = append(olderSubCommittee, index)
		}
	}
	for _, index := range newerCommittee {
		if index%params.BeaconConfig().EpochsPerShardPeriod > epoch%params.BeaconConfig().EpochsPerShardPeriod {
			newerSubCommittee = append(newerSubCommittee, index)
		}
	}

	return append(olderSubCommittee, newerSubCommittee...), nil
}

// ShardProposerIndex returns the shard proposer index of a given shard slot.
//
// Spec pseudocode definition:
//  def get_shard_proposer_index(state: BeaconState, shard: Shard, slot: ShardSlot) -> ValidatorIndex:
//    epoch = get_current_epoch(state)
//    active_indices = [i for i in get_shard_committee(state, shard, epoch) if is_active_validator(state.validators[i], epoch)]
//    seed = hash(get_seed(state, epoch) + int_to_bytes(slot, length=8) + int_to_bytes(shard, length=8))
//    compute_proposer_index(state, active_indices, seed)
func ShardProposerIndex(state *pb.BeaconState, shard uint64, slot uint64) (uint64, error) {
	epoch := helpers.CurrentEpoch(state)
	shardCommittee, err := ShardCommittee(state, shard, epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get shard committee")
	}
	activeIndices := make([]uint64, 0, len(shardCommittee))
	for _, index := range shardCommittee {
		if helpers.IsActiveValidator(state.Validators[index], epoch) {
			activeIndices = append(activeIndices, index)
		}
	}
	s, err := helpers.Seed(state, epoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get seed")
	}
	slotBytes := bytesutil.ToBytes(slot, 8)
	shardBytes := bytesutil.ToBytes(shard, 8)
	seed := s[:]
	seed = append(seed, slotBytes...)
	seed = append(seed, shardBytes...)

	// TODO(1371): https://github.com/ethereum/eth2.0-specs/pull/1371/files
	// compute_proposer_index is getting implemented in #1371 eth2 repo.
	return 0, fmt.Errorf("not implemented")
}
