package validator

import (
	"math"
	"slices"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/operations/attestations/kv"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// computeOnChainAggregate constructs an on chain final aggregate form a list of network aggregates with equal attestation data.
// It assumes that each network aggregate has exactly one committee bit set.
// The spec defines how to construct a final aggregate from one set of network aggregates, but computeOnChainAggregate does this
// for any number of such sets (these sets are bundled together in the map argument).
//
// Spec definition:
//
//	def compute_on_chain_aggregate(network_aggregates: Sequence[Attestation]) -> Attestation:
//		aggregates = sorted(network_aggregates, key=lambda a: get_committee_indices(a.committee_bits)[0])
//
//		data = aggregates[0].data
//		aggregation_bits = Bitlist[MAX_VALIDATORS_PER_COMMITTEE * MAX_COMMITTEES_PER_SLOT]()
//		for a in aggregates:
//			for b in a.aggregation_bits:
//				aggregation_bits.append(b)
//
//		signature = bls.Aggregate([a.signature for a in aggregates])
//
//		committee_indices = [get_committee_indices(a.committee_bits)[0] for a in aggregates]
//		committee_flags = [(index in committee_indices) for index in range(0, MAX_COMMITTEES_PER_SLOT)]
//		committee_bits = Bitvector[MAX_COMMITTEES_PER_SLOT](committee_flags)
//
//		return Attestation(
//			aggregation_bits=aggregation_bits,
//			data=data,
//			committee_bits=committee_bits,
//			signature=signature,
//		)
func computeOnChainAggregate(aggregates map[kv.AttestationId][]ethpb.Att) ([]ethpb.Att, error) {
	// Digest is the attestation data root. The incoming map has attestations for the same root
	// but different committee indices under different keys. We create a new map where the digest is the key
	// so that all attestations for the same root are under one key.
	aggsByDigest := make(map[[32]byte][]ethpb.Att, 0)
	for id, aggs := range aggregates {
		existing, ok := aggsByDigest[id.Digest]
		if ok {
			aggsByDigest[id.Digest] = append(existing, aggs...)
		} else {
			aggsByDigest[id.Digest] = aggs
		}
	}

	result := make([]ethpb.Att, 0)

	for _, aggs := range aggsByDigest {
		slices.SortFunc(aggs, func(a, b ethpb.Att) int {
			return a.GetCommitteeBitsVal().BitIndices()[0] - b.GetCommitteeBitsVal().BitIndices()[0]
		})

		sigs := make([]bls.Signature, len(aggs))
		committeeIndices := make([]primitives.CommitteeIndex, len(aggs))
		aggBitsIndices := make([]uint64, 0)
		aggBitsOffset := uint64(0)
		var err error
		for i, a := range aggs {
			for _, bi := range a.GetAggregationBits().BitIndices() {
				aggBitsIndices = append(aggBitsIndices, uint64(bi)+aggBitsOffset)
			}
			sigs[i], err = bls.SignatureFromBytes(a.GetSignature())
			if err != nil {
				return nil, err
			}
			committeeIndices[i] = helpers.CommitteeIndices(a.GetCommitteeBitsVal())[0]

			aggBitsOffset += a.GetAggregationBits().Len()
		}

		aggregationBits := bitfield.NewBitlist(uint64(aggBitsOffset))
		for _, bi := range aggBitsIndices {
			aggregationBits.SetBitAt(uint64(bi), true)
		}

		// TODO: hack
		committeeBits := make([]byte, int(math.Ceil(float64(params.BeaconConfig().MaxCommitteesPerSlot)/float64(8))))

		att := &ethpb.AttestationElectra{
			AggregationBits: aggregationBits,
			Data:            aggs[0].GetData(),
			CommitteeBits:   committeeBits,
			Signature:       bls.AggregateSignatures(sigs).Marshal(),
		}
		for _, ci := range committeeIndices {
			att.CommitteeBits.SetBitAt(uint64(ci), true)
		}

		result = append(result, att)
	}

	return result, nil
}
