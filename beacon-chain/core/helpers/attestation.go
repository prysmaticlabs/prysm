package helpers

import (
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	// ErrAttestationAggregationBitsOverlap is returned when two attestations aggregation
	// bits overlap with each other.
	ErrAttestationAggregationBitsOverlap = errors.New("overlapping aggregation bits")

	// ErrAttestationAggregationBitsDifferentLen is returned when two attestation aggregation bits
	// have different lengths.
	ErrAttestationAggregationBitsDifferentLen = errors.New("different bitlist lengths")
)

// AggregateAttestations such that the minimal number of attestations are returned.
// Note: this is currently a naive implementation to the order of O(n^2).
func AggregateAttestations(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) <= 1 {
		return atts, nil
	}

	// Naive aggregation. O(n^2) time.
	for i, a := range atts {
		if i >= len(atts) {
			break
		}
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]
			if a.AggregationBits.Len() == b.AggregationBits.Len() && !a.AggregationBits.Overlaps(b.AggregationBits) {
				var err error
				a, err = AggregateAttestation(a, b)
				if err != nil {
					return nil, err
				}
				// Delete b
				atts = append(atts[:j], atts[j+1:]...)
				j--
				atts[i] = a
			}
		}
	}

	// Naive deduplication of identical aggregations. O(n^2) time.
	for i, a := range atts {
		for j := i + 1; j < len(atts); j++ {
			b := atts[j]

			if a.AggregationBits.Len() != b.AggregationBits.Len() {
				continue
			}

			if a.AggregationBits.Contains(b.AggregationBits) {
				// If b is fully contained in a, then b can be removed.
				atts = append(atts[:j], atts[j+1:]...)
				j--
			} else if b.AggregationBits.Contains(a.AggregationBits) {
				// if a is fully contained in b, then a can be removed.
				atts = append(atts[:i], atts[i+1:]...)
				i--
				break // Stop the inner loop, advance a.
			}
		}
	}

	return atts, nil
}

// BLS aggregate signature aliases for testing / benchmark substitution. These methods are
// significantly more expensive than the inner logic of AggregateAttestations so they must be
// substituted for benchmarks which analyze AggregateAttestations.
var aggregateSignatures = bls.AggregateSignatures
var signatureFromBytes = bls.SignatureFromBytes

// AggregateAttestation aggregates attestations a1 and a2 together.
func AggregateAttestation(a1 *ethpb.Attestation, a2 *ethpb.Attestation) (*ethpb.Attestation, error) {
	if a1.AggregationBits.Len() != a2.AggregationBits.Len() {
		return nil, ErrAttestationAggregationBitsDifferentLen
	}
	if a1.AggregationBits.Overlaps(a2.AggregationBits) {
		return nil, ErrAttestationAggregationBitsOverlap
	}

	baseAtt := stateTrie.CopyAttestation(a1)
	newAtt := stateTrie.CopyAttestation(a2)
	if newAtt.AggregationBits.Count() > baseAtt.AggregationBits.Count() {
		baseAtt, newAtt = newAtt, baseAtt
	}

	if baseAtt.AggregationBits.Contains(newAtt.AggregationBits) {
		return baseAtt, nil
	}

	newBits := baseAtt.AggregationBits.Or(newAtt.AggregationBits)
	newSig, err := signatureFromBytes(newAtt.Signature)
	if err != nil {
		return nil, err
	}
	baseSig, err := signatureFromBytes(baseAtt.Signature)
	if err != nil {
		return nil, err
	}

	aggregatedSig := aggregateSignatures([]*bls.Signature{baseSig, newSig})
	baseAtt.Signature = aggregatedSig.Marshal()
	baseAtt.AggregationBits = newBits

	return baseAtt, nil
}

// SlotSignature returns the signed signature of the hash tree root of input slot.
//
// Spec pseudocode definition:
//   def get_slot_signature(state: BeaconState, slot: Slot, privkey: int) -> BLSSignature:
//    domain = get_domain(state, DOMAIN_BEACON_ATTESTER, compute_epoch_at_slot(slot))
//    return bls_sign(privkey, hash_tree_root(slot), domain)
func SlotSignature(state *stateTrie.BeaconState, slot uint64, privKey *bls.SecretKey) (*bls.Signature, error) {
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
func AggregateSignature(attestations []*ethpb.Attestation) (*bls.Signature, error) {
	sigs := make([]*bls.Signature, len(attestations))
	var err error
	for i := 0; i < len(sigs); i++ {
		sigs[i], err = signatureFromBytes(attestations[i].Signature)
		if err != nil {
			return nil, err
		}
	}
	return aggregateSignatures(sigs), nil
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
