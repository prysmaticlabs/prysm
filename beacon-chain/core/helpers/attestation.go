package helpers

import (
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// AttestationDataSlot returns current slot of AttestationData for given state
//
// Spec pseudocode definition:
// def get_attestation_data_slot(state: BeaconState, data: AttestationData) -> Slot:
// committee_count = get_epoch_committee_count(state, data.target_epoch)
// offset = (data.crosslink.shard + SHARD_COUNT - get_epoch_start_shard(state, data.target_epoch)) % SHARD_COUNT
// return get_epoch_start_slot(data.target_epoch) + offset // (committee_count // SLOTS_PER_EPOCH)
func AttestationDataSlot(state *pb.BeaconState, data *pb.AttestationData) uint64 {
	commiteeCount := EpochCommitteeCount(state, data.TargetEpoch)
	offset := (data.Crosslink.Shard + params.BeaconConfig().ShardCount -
		EpochStartShard(state, data.TargetEpoch)) % params.BeaconConfig().ShardCount

	return StartSlot(data.TargetEpoch) + offset/(commiteeCount/params.BeaconConfig().SlotsPerEpoch)
}
