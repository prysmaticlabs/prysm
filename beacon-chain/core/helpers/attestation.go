package helpers

import (
	"encoding/binary"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var (
	// ErrAttestationDataSlotNilState is returned when a nil state argument
	// is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilState = errors.New("nil state provided for AttestationDataSlot")
	// ErrAttestationDataSlotNilData is returned when a nil attestation data
	// argument is provided to AttestationDataSlot.
	ErrAttestationDataSlotNilData = errors.New("nil data provided for AttestationDataSlot")
	// ErrAttestationAggregationBitsOverlap is returned when two attestations aggregation
	// bits overlap with each other.
	ErrAttestationAggregationBitsOverlap = errors.New("overlapping aggregation bits")
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
			if !a.AggregationBits.Overlaps(b.AggregationBits) {
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
	if a1.AggregationBits.Overlaps(a2.AggregationBits) {
		return nil, ErrAttestationAggregationBitsOverlap
	}

	baseAtt := proto.Clone(a1).(*ethpb.Attestation)
	newAtt := proto.Clone(a2).(*ethpb.Attestation)
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
//   def slot_signature(state: BeaconState, slot: Slot, privkey: int) -> BLSSignature:
//    domain = get_domain(state, DOMAIN_BEACON_ATTESTER, compute_epoch_at_slot(slot))
//    return bls_sign(privkey, hash_tree_root(slot), domain)
func SlotSignature(state *pb.BeaconState, slot uint64, privKey *bls.SecretKey) (*bls.Signature, error) {
	d := Domain(state.Fork, CurrentEpoch(state), params.BeaconConfig().DomainBeaconAttester)
	s, err := ssz.HashTreeRoot(slot)
	if err != nil {
		return nil, err
	}
	return privKey.Sign(s[:], d), nil
}

// IsAggregator returns true if the signature is from the input validator.
//
// Spec pseudocode definition:
//   def is_aggregator(state: BeaconState, slot: Slot, index: CommitteeIndex, slot_signature: BLSSignature) -> bool:
//    committee = get_beacon_committee(state, slot, index)
//    modulo = max(1, len(committee) // TARGET_AGGREGATORS_PER_COMMITTEE)
//    return bytes_to_int(hash(slot_signature)[0:8]) % modulo == 0
func IsAggregator(state *pb.BeaconState, slot uint64, index uint64, slotSig *bls.Signature) (bool, error) {
	committee, err := BeaconCommittee(state, slot, index)
	if err != nil {
		return false, err
	}
	modulo := uint64(1)
	if len(committee)/int(params.BeaconConfig().TargetAggregatorsPerCommittee) > 1 {
		modulo = uint64(len(committee)) / params.BeaconConfig().TargetAggregatorsPerCommittee
	}

	b := hashutil.Hash(slotSig.Marshal())
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
