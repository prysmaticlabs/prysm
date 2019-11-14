package helper

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// OnlineIndices returns the validator indices that are online from beacon state.
//
// Spec pseudocode definition:
//   def get_online_indices(state: BeaconState) -> Set[ValidatorIndex]:
//    active_validators = get_active_validator_indices(state, get_current_epoch(state))
//    return set([i for i in active_validators if state.online_countdown[i] != 0])
func OnlineIndices(state *pb.BeaconState) ([]uint64, error) {
	indices, err := helpers.ActiveValidatorIndices(state, helpers.CurrentEpoch(state))
	if err != nil {
		return nil, err
	}
	onlineIndices := make([]uint64, 0, len(indices))
	for _, i := range indices {
		if !bytes.Equal(state.OnlineCountdown[i], []byte{0}) {
			onlineIndices = append(onlineIndices, i)
		}
	}

	return onlineIndices, nil
}

// ShardProposerIndex returns the shard proposer index of a given slot and shard.
//
// Spec pseudocode definition:
//   def get_shard_proposer_index(beacon_state: BeaconState, slot: Slot, shard: Shard) -> ValidatorIndex:
//    committee = get_shard_committee(beacon_state, slot_to_epoch(slot), shard)
//    r = bytes_to_int(get_seed(beacon_state, get_current_epoch(state), DOMAIN_SHARD_COMMITTEE)[:8])
//    return committee[r % len(committee)]
func ShardProposerIndex(state *pb.BeaconState, slot uint64, shard uint64) (uint64, error) {
	committee, err := ShardCommittee(state, helpers.SlotToEpoch(slot), shard)
	seed, err := helpers.Seed(state, helpers.CurrentEpoch(state), params.BeaconConfig().DomainShardProposal)
	if err != nil {
		return 0, err
	}
	propoerIndex := int(bytesutil.FromBytes8(seed[:8]))

	return committee[propoerIndex%len(committee)], nil
}
