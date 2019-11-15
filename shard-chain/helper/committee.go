package helper

import (
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// PackCompactValidator packs validator index, slashed status and compressed balance into a single uint value.
//
// Spec pseudocode definition:
//   def pack_compact_validator(index: int, slashed: bool, balance_in_increments: int) -> int:
//    """
//    Creates a compact validator object representing index, slashed status, and compressed balance.
//    Takes as input balance-in-increments (// EFFECTIVE_BALANCE_INCREMENT) to preserve symmetry with
//    the unpacking function.
//    """
//    return (index << 16) + (slashed << 15) + balance_in_increments
func PackCompactValidator(index uint64, slashed bool, balanceIncrements uint64) uint64 {
	if slashed {
		return (index << 16) + (1 << 15) + balanceIncrements
	}
	return (index << 16) + (0 << 15) + balanceIncrements
}

// CommitteeToCompactCommittee converts a committee object to compact committee object.
//
// Spec pseudocode definition:
//   def committee_to_compact_committee(state: BeaconState, committee: Sequence[ValidatorIndex]) -> CompactCommittee:
//    """
//    Given a state and a list of validator indices, outputs the CompactCommittee representing them.
//    """
//    validators = [state.validators[i] for i in committee]
//    compact_validators = [
//        pack_compact_validator(i, v.slashed, v.effective_balance // EFFECTIVE_BALANCE_INCREMENT)
//        for i, v in zip(committee, validators)
//    ]
//    pubkeys = [v.pubkey for v in validators]
//    return CompactCommittee(pubkeys=pubkeys, compact_validators=compact_validators)
func CommitteeToCompactCommittee(state *pb.BeaconState, committee []uint64) *ethpb.CompactCommittee {
	compactValidators := make([]uint64, len(committee))
	pubKeys := make([][]byte, len(committee))

	for i := 0; i < len(committee); i++ {
		v := state.Validators[committee[i]]
		compactValidators[i] = PackCompactValidator(committee[i], v.Slashed, v.EffectiveBalance/params.BeaconConfig().EffectiveBalanceIncrement)
		pubKeys[i] = v.PublicKey
	}

	return &ethpb.CompactCommittee{CompactValidators: compactValidators, Pubkeys: pubKeys}
}

// ShardCommittee returns the shard committee of the given epoch and shard.
//
// Spec pseudocode definition:
//   def get_shard_committee(beacon_state: BeaconState, epoch: Epoch, shard: Shard) -> Sequence[ValidatorIndex]:
//    source_epoch = epoch - epoch % SHARD_COMMITTEE_PERIOD
//    if source_epoch > 0:
//        source_epoch -= SHARD_COMMITTEE_PERIOD
//    active_validator_indices = get_active_validator_indices(beacon_state, source_epoch)
//    seed = get_seed(beacon_state, source_epoch, DOMAIN_SHARD_COMMITTEE)
//    return compute_committee(active_validator_indices, seed, 0, ACTIVE_SHARDS)
func ShardCommittee(state *pb.BeaconState, epoch uint64, shard uint64) ([]uint64, error) {
	sourceEpoch := epoch - epoch%params.BeaconConfig().ShardCommitteePeriod
	if sourceEpoch >= params.BeaconConfig().ShardCommitteePeriod {
		sourceEpoch -= params.BeaconConfig().ShardCommitteePeriod
	}

	indices, err := helpers.ActiveValidatorIndices(state, sourceEpoch)
	if err != nil {
		return nil, err
	}

	seed, err := helpers.Seed(state, sourceEpoch, params.BeaconConfig().DomainShardCommittee)
	if err != nil {
		return nil, err
	}

	return helpers.ComputeCommittee(indices, seed, 0, params.BeaconConfig().ActiveShards)
}

// LightClientCommittee returns the light client committee of the given epoch.
//
// Spec pseudocode definition:
//   def get_light_client_committee(beacon_state: BeaconState, epoch: Epoch) -> Sequence[ValidatorIndex]:
//    source_epoch = epoch - epoch % LIGHT_CLIENT_COMMITTEE_PERIOD
//    if source_epoch > 0:
//        source_epoch -= LIGHT_CLIENT_COMMITTEE_PERIOD
//    active_validator_indices = get_active_validator_indices(beacon_state, source_epoch)
//    seed = get_seed(beacon_state, source_epoch, DOMAIN_SHARD_LIGHT_CLIENT)
//    return compute_committee(active_validator_indices, seed, 0, ACTIVE_SHARDS)[:TARGET_COMMITTEE_SIZE]
func LightClientCommittee(state *pb.BeaconState, epoch uint64) ([]uint64, error) {
	sourceEpoch := epoch - epoch%params.BeaconConfig().ShardCommitteePeriod
	if sourceEpoch >= params.BeaconConfig().ShardCommitteePeriod {
		sourceEpoch -= params.BeaconConfig().ShardCommitteePeriod
	}

	indices, err := helpers.ActiveValidatorIndices(state, sourceEpoch)
	if err != nil {
		return nil, err
	}

	seed, err := helpers.Seed(state, sourceEpoch, params.BeaconConfig().DomainShardLightClient)
	if err != nil {
		return nil, err
	}

	committee, err := helpers.ComputeCommittee(indices, seed, 0, params.BeaconConfig().ActiveShards)
	if err != nil {
		return nil, err
	}

	return committee[:params.BeaconConfig().TargetCommitteeSize], nil
}
