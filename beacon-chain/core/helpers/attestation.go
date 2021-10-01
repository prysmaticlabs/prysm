package helpers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/time"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// ValidateNilAttestation checks if any composite field of input attestation is nil.
// Access to these nil fields will result in run time panic,
// it is recommended to run these checks as first line of defense.
func ValidateNilAttestation(attestation *ethpb.Attestation) error {
	if attestation == nil {
		return errors.New("attestation can't be nil")
	}
	if attestation.Data == nil {
		return errors.New("attestation's data can't be nil")
	}
	if attestation.Data.Source == nil {
		return errors.New("attestation's source can't be nil")
	}
	if attestation.Data.Target == nil {
		return errors.New("attestation's target can't be nil")
	}
	if attestation.AggregationBits == nil {
		return errors.New("attestation's bitfield can't be nil")
	}
	return nil
}

// ValidateSlotTargetEpoch checks if attestation data's epoch matches target checkpoint's epoch.
// It is recommended to run `ValidateNilAttestation` first to ensure `data.Target` can't be nil.
func ValidateSlotTargetEpoch(data *ethpb.AttestationData) error {
	if slots.ToEpoch(data.Slot) != data.Target.Epoch {
		return fmt.Errorf("slot %d does not match target epoch %d", data.Slot, data.Target.Epoch)
	}
	return nil
}

// IsAggregator returns true if the signature is from the input validator. The committee
// count is provided as an argument rather than imported implementation from spec. Having
// committee count as an argument allows cheaper computation at run time.
//
// Spec pseudocode definition:
//   def is_aggregator(state: BeaconState, slot: Slot, index: CommitteeIndex, slot_signature: BLSSignature) -> bool:
//    committee = get_beacon_committee(state, slot, index)
//    modulo = max(1, len(committee) // TARGET_AGGREGATORS_PER_COMMITTEE)
//    return bytes_to_uint64(hash(slot_signature)[0:8]) % modulo == 0
func IsAggregator(committeeCount uint64, slotSig []byte) (bool, error) {
	modulo := uint64(1)
	if committeeCount/params.BeaconConfig().TargetAggregatorsPerCommittee > 1 {
		modulo = committeeCount / params.BeaconConfig().TargetAggregatorsPerCommittee
	}

	b := hash.Hash(slotSig)
	return binary.LittleEndian.Uint64(b[:8])%modulo == 0, nil
}

// AggregateSignature returns the aggregated signature of the input attestations.
//
// Spec pseudocode definition:
//   def get_aggregate_signature(attestations: Sequence[Attestation]) -> BLSSignature:
//    signatures = [attestation.signature for attestation in attestations]
//    return bls.Aggregate(signatures)
func AggregateSignature(attestations []*ethpb.Attestation) (bls.Signature, error) {
	sigs := make([]bls.Signature, len(attestations))
	var err error
	for i := 0; i < len(sigs); i++ {
		sigs[i], err = bls.SignatureFromBytes(attestations[i].Signature)
		if err != nil {
			return nil, err
		}
	}
	return bls.AggregateSignatures(sigs), nil
}

// IsAggregated returns true if the attestation is an aggregated attestation,
// false otherwise.
func IsAggregated(attestation *ethpb.Attestation) bool {
	return attestation.AggregationBits.Count() > 1
}

// ComputeSubnetForAttestation returns the subnet for which the provided attestation will be broadcasted to.
// This differs from the spec definition by instead passing in the active validators indices in the attestation's
// given epoch.
//
// Spec pseudocode definition:
// def compute_subnet_for_attestation(committees_per_slot: uint64, slot: Slot, committee_index: CommitteeIndex) -> uint64:
//    """
//    Compute the correct subnet for an attestation for Phase 0.
//    Note, this mimics expected future behavior where attestations will be mapped to their shard subnet.
//    """
//    slots_since_epoch_start = uint64(slot % SLOTS_PER_EPOCH)
//    committees_since_epoch_start = committees_per_slot * slots_since_epoch_start
//
//    return uint64((committees_since_epoch_start + committee_index) % ATTESTATION_SUBNET_COUNT)
func ComputeSubnetForAttestation(activeValCount uint64, att *ethpb.Attestation) uint64 {
	return ComputeSubnetFromCommitteeAndSlot(activeValCount, att.Data.CommitteeIndex, att.Data.Slot)
}

// ComputeSubnetFromCommitteeAndSlot is a flattened version of ComputeSubnetForAttestation where we only pass in
// the relevant fields from the attestation as function arguments.
//
// Spec pseudocode definition:
// def compute_subnet_for_attestation(committees_per_slot: uint64, slot: Slot, committee_index: CommitteeIndex) -> uint64:
//    """
//    Compute the correct subnet for an attestation for Phase 0.
//    Note, this mimics expected future behavior where attestations will be mapped to their shard subnet.
//    """
//    slots_since_epoch_start = uint64(slot % SLOTS_PER_EPOCH)
//    committees_since_epoch_start = committees_per_slot * slots_since_epoch_start
//
//    return uint64((committees_since_epoch_start + committee_index) % ATTESTATION_SUBNET_COUNT)
func ComputeSubnetFromCommitteeAndSlot(activeValCount uint64, comIdx types.CommitteeIndex, attSlot types.Slot) uint64 {
	slotSinceStart := slots.SinceEpochStarts(attSlot)
	comCount := SlotCommitteeCount(activeValCount)
	commsSinceStart := uint64(slotSinceStart.Mul(comCount))
	computedSubnet := (commsSinceStart + uint64(comIdx)) % params.BeaconNetworkConfig().AttestationSubnetCount
	return computedSubnet
}

// ValidateAttestationTime Validates that the incoming attestation is in the desired time range.
// An attestation is valid only if received within the last ATTESTATION_PROPAGATION_SLOT_RANGE
// slots.
//
// Example:
//   ATTESTATION_PROPAGATION_SLOT_RANGE = 5
//   clockDisparity = 24 seconds
//   current_slot = 100
//   invalid_attestation_slot = 92
//   invalid_attestation_slot = 103
//   valid_attestation_slot = 98
//   valid_attestation_slot = 101
// In the attestation must be within the range of 95 to 102 in the example above.
func ValidateAttestationTime(attSlot types.Slot, genesisTime time.Time, clockDisparity time.Duration) error {
	if err := slots.ValidateClock(attSlot, uint64(genesisTime.Unix())); err != nil {
		return err
	}
	attTime, err := slots.ToTime(uint64(genesisTime.Unix()), attSlot)
	if err != nil {
		return err
	}
	currentSlot := slots.Since(genesisTime)

	// When receiving an attestation, it can be from the future.
	// so the upper bounds is set to now + clockDisparity(SECONDS_PER_SLOT * 2).
	// But when sending an attestation, it should not be in future slot.
	// so the upper bounds is set to now + clockDisparity(MAXIMUM_GOSSIP_CLOCK_DISPARITY).
	upperBounds := prysmTime.Now().Add(clockDisparity)

	// An attestation cannot be older than the current slot - attestation propagation slot range
	// with a minor tolerance for peer clock disparity.
	lowerBoundsSlot := types.Slot(0)
	if currentSlot > params.BeaconNetworkConfig().AttestationPropagationSlotRange {
		lowerBoundsSlot = currentSlot - params.BeaconNetworkConfig().AttestationPropagationSlotRange
	}
	lowerTime, err := slots.ToTime(uint64(genesisTime.Unix()), lowerBoundsSlot)
	if err != nil {
		return err
	}
	lowerBounds := lowerTime.Add(-clockDisparity)

	// Verify attestation slot within the time range.
	if attTime.Before(lowerBounds) || attTime.After(upperBounds) {
		return fmt.Errorf(
			"attestation slot %d not within attestation propagation range of %d to %d (current slot)",
			attSlot,
			lowerBoundsSlot,
			currentSlot,
		)
	}
	return nil
}

// VerifyCheckpointEpoch is within current epoch and previous epoch
// with respect to current time. Returns true if it's within, false if it's not.
func VerifyCheckpointEpoch(c *ethpb.Checkpoint, genesis time.Time) bool {
	now := uint64(prysmTime.Now().Unix())
	genesisTime := uint64(genesis.Unix())
	currentSlot := types.Slot((now - genesisTime) / params.BeaconConfig().SecondsPerSlot)
	currentEpoch := slots.ToEpoch(currentSlot)

	var prevEpoch types.Epoch
	if currentEpoch > 1 {
		prevEpoch = currentEpoch - 1
	}

	if c.Epoch != prevEpoch && c.Epoch != currentEpoch {
		return false
	}

	return true
}
