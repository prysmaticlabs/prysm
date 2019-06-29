package helpers

import (
	"errors"
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	ErrAttestationDataSlotNilState = errors.New("nil state provided for AttestationDataSlot")
	ErrAttestationDataSlotNilData  = errors.New("nil data provided for AttestationDataSlot")
)

// AttestationDataSlot returns current slot of AttestationData for given state
//
// Spec pseudocode definition:
//   def get_attestation_data_slot(state: BeaconState, data: AttestationData) -> Slot:
//     committee_count = get_epoch_committee_count(state, data.target_epoch)
//     offset = (data.crosslink.shard + SHARD_COUNT - get_epoch_start_shard(state, data.target_epoch)) % SHARD_COUNT
//     return get_epoch_start_slot(data.target_epoch) + offset // (committee_count // SLOTS_PER_EPOCH)
func AttestationDataSlot(state *pb.BeaconState, data *pb.AttestationData) (uint64, error) {
	if state == nil {
		return 0, ErrAttestationDataSlotNilState
	}
	if data == nil {
		return 0, ErrAttestationDataSlotNilData
	}
	committeeCount, err := EpochCommitteeCount(state, data.TargetEpoch)
	if err != nil {
		return 0, err
	}

	epochStartShardNumber, err := EpochStartShard(state, data.TargetEpoch)
	if err != nil { // This should never happen if EpochCommitteeCount was successful
		return 0, fmt.Errorf("could not determine epoch start shard: %v", err)
	}
	offset := (data.Crosslink.Shard + params.BeaconConfig().ShardCount -
		epochStartShardNumber) % params.BeaconConfig().ShardCount

	return StartSlot(data.TargetEpoch) + offset/(committeeCount/params.BeaconConfig().SlotsPerEpoch), nil
}
