package helpers

import (
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"

	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	// ErrAttestationDataSlotNilState is returned when a nil state argument
	// is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilState = errors.New("nil state provided for AttestationDataSlot")
	// ErrAttestationDataSlotNilData is returned when a nil attestation data
	// argument is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilData = errors.New("nil data provided for AttestationDataSlot")
)

// AttestationDataSlot returns current slot of AttestationData for given state
//
// Spec pseudocode definition:
//   def get_attestation_data_slot(state: BeaconState, data: AttestationData) -> Slot:
//    """
//    Return the slot corresponding to the attestation ``data``.
//    """
//    committee_count = get_committee_count(state, data.target.epoch)
//    offset = (data.crosslink.shard + SHARD_COUNT - get_start_shard(state, data.target.epoch)) % SHARD_COUNT
//    return Slot(compute_start_slot_of_epoch(data.target.epoch) + offset // (committee_count // SLOTS_PER_EPOCH))
func AttestationDataSlot(state *pb.BeaconState, data *ethpb.AttestationData) (uint64, error) {
	if state == nil {
		return 0, ErrAttestationDataSlotNilState
	}
	if data == nil {
		return 0, ErrAttestationDataSlotNilData
	}

	committeeCount, err := CommitteeCount(state, data.Target.Epoch)
	if err != nil {
		return 0, err
	}

	epochStartShardNumber, err := StartShard(state, data.Target.Epoch)
	if err != nil { // This should never happen if CommitteeCount was successful
		return 0, errors.Wrap(err, "could not determine epoch start shard")
	}
	offset := (data.Crosslink.Shard + params.BeaconConfig().ShardCount -
		epochStartShardNumber) % params.BeaconConfig().ShardCount

	return StartSlot(data.Target.Epoch) + (offset / (committeeCount / params.BeaconConfig().SlotsPerEpoch)), nil
}
