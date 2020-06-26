package helpers

import (
	"encoding/binary"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
)

// SlotSignature returns the signed signature of the hash tree root of input slot.
//
// Spec pseudocode definition:
//   def get_slot_signature(state: BeaconState, slot: Slot, privkey: int) -> BLSSignature:
//    domain = get_domain(state, DOMAIN_SELECTION_PROOF, compute_epoch_at_slot(slot))
//    signing_root = compute_signing_root(slot, domain)
//    return bls.Sign(privkey, signing_root)
func SlotSignature(state *stateTrie.BeaconState, slot uint64, privKey bls.SecretKey) (bls.Signature, error) {
	d, err := Domain(state.Fork(), CurrentEpoch(state), params.BeaconConfig().DomainBeaconAttester, state.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	s, err := ComputeSigningRoot(slot, d)
	if err != nil {
		return nil, err
	}
	return privKey.Sign(s[:]), nil
}

// IsAggregator returns true if the signature is from the input validator. The committee
// count is provided as an argument rather than direct implementation from spec. Having
// committee count as an argument allows cheaper computation at run time.
//
// Spec pseudocode definition:
//   def is_aggregator(state: BeaconState, slot: Slot, index: CommitteeIndex, slot_signature: BLSSignature) -> bool:
//    committee = get_beacon_committee(state, slot, index)
//    modulo = max(1, len(committee) // TARGET_AGGREGATORS_PER_COMMITTEE)
//    return bytes_to_int(hash(slot_signature)[0:8]) % modulo == 0
func IsAggregator(committeeCount uint64, slotSig []byte) (bool, error) {
	modulo := uint64(1)
	if committeeCount/params.BeaconConfig().TargetAggregatorsPerCommittee > 1 {
		modulo = committeeCount / params.BeaconConfig().TargetAggregatorsPerCommittee
	}

	b := hashutil.Hash(slotSig)
	return binary.LittleEndian.Uint64(b[:8])%modulo == 0, nil
}

// AggregateSignature returns the aggregated signature of the input attestations.
//
// Spec pseudocode definition:
//   def get_aggregate_signature(attestations: Sequence[Attestation]) -> BLSSignature:
//    signatures = [attestation.signature for attestation in attestations]
//    return bls_aggregate_signatures(signatures)
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
// def compute_subnet_for_attestation(state: BeaconState, attestation: Attestation) -> uint64:
//    """
//    Compute the correct subnet for an attestation for Phase 0.
//    Note, this mimics expected Phase 1 behavior where attestations will be mapped to their shard subnet.
//    """
//    slots_since_epoch_start = attestation.data.slot % SLOTS_PER_EPOCH
//    committees_since_epoch_start = get_committee_count_at_slot(state, attestation.data.slot) * slots_since_epoch_start
//    return (committees_since_epoch_start + attestation.data.index) % ATTESTATION_SUBNET_COUNT
func ComputeSubnetForAttestation(activeValCount uint64, att *ethpb.Attestation) uint64 {
	return ComputeSubnetFromCommitteeAndSlot(activeValCount, att.Data.CommitteeIndex, att.Data.Slot)
}

// ComputeSubnetFromCommitteeAndSlot is a flattened version of ComputeSubnetForAttestation where we only pass in
// the relevant fields from the attestation as function arguments.
//
// Spec pseudocode definition:
// def compute_subnet_for_attestation(state: BeaconState, attestation: Attestation) -> uint64:
//    """
//    Compute the correct subnet for an attestation for Phase 0.
//    Note, this mimics expected Phase 1 behavior where attestations will be mapped to their shard subnet.
//    """
//    slots_since_epoch_start = attestation.data.slot % SLOTS_PER_EPOCH
//    committees_since_epoch_start = get_committee_count_at_slot(state, attestation.data.slot) * slots_since_epoch_start
//    return (committees_since_epoch_start + attestation.data.index) % ATTESTATION_SUBNET_COUNT
func ComputeSubnetFromCommitteeAndSlot(activeValCount, comIdx, attSlot uint64) uint64 {
	slotSinceStart := SlotsSinceEpochStarts(attSlot)
	comCount := SlotCommitteeCount(activeValCount)
	commsSinceStart := comCount * slotSinceStart
	computedSubnet := (commsSinceStart + comIdx) % params.BeaconNetworkConfig().AttestationSubnetCount
	return computedSubnet
}

// Validates that the incoming attestation is in the desired time range. An attestation
// is valid only if received within the last ATTESTATION_PROPAGATION_SLOT_RANGE slots.
//
// Example:
//   ATTESTATION_PROPAGATION_SLOT_RANGE = 5
//   current_slot = 100
//   invalid_attestation_slot = 92
//   invalid_attestation_slot = 101
//   valid_attestation_slot = 98
// In the attestation must be within the range of 95 to 100 in the example above.
func ValidateAttestationTime(attSlot uint64, genesisTime time.Time) error {
	attTime := genesisTime.Add(time.Duration(attSlot*params.BeaconConfig().SecondsPerSlot) * time.Second)
	currentSlot := SlotsSince(genesisTime)

	// A clock disparity allows for minor tolerances outside of the expected range. This value is
	// usually small, less than 1 second.
	clockDisparity := params.BeaconNetworkConfig().MaximumGossipClockDisparity

	// An attestation cannot be from the future, so the upper bounds is set to now, with a minor
	// tolerance for peer clock disparity.
	upperBounds := roughtime.Now().Add(clockDisparity)

	// An attestation cannot be older than the current slot - attestation propagation slot range
	// with a minor tolerance for peer clock disparity.
	lowerBoundsSlot := uint64(0)
	if currentSlot > params.BeaconNetworkConfig().AttestationPropagationSlotRange {
		lowerBoundsSlot = currentSlot - params.BeaconNetworkConfig().AttestationPropagationSlotRange
	}
	lowerBounds := genesisTime.Add(
		time.Duration(lowerBoundsSlot*params.BeaconConfig().SecondsPerSlot) * time.Second,
	).Add(-clockDisparity)

	// Verify attestation slot within the time range.
	if attTime.Before(lowerBounds) || attTime.After(upperBounds) {
		return fmt.Errorf(
			"attestation slot %d not within attestation propagation range of %d to %d (current slot)",
			attSlot,
			currentSlot-params.BeaconNetworkConfig().AttestationPropagationSlotRange,
			currentSlot,
		)
	}
	return nil
}
