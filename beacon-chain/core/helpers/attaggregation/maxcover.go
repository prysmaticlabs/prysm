package attaggregation

import (
	"errors"
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// MaxCoverAggregation is a strategy based on Maximum Coverage greedy algorithm.
const MaxCoverAggregation AttestationAggregationStrategy = "max_cover"

// ErrInvalidAttestationCount is returned when insufficient number
// of attestations is provided for aggregation.
var ErrInvalidAttestationCount = errors.New("invalid number of attestations")

// MaxCoverAttestationAggregation relies on Maximum Coverage greedy algorithm for aggregation.
//
// For full analysis or running time, see "Analysis of the Greedy Approach in Problems of
// Maximum k-Coverage" by Hochbaum and Pathria.
func MaxCoverAttestationAggregation(atts []*ethpb.Attestation) ([]*ethpb.Attestation, error) {
	if len(atts) < 2 {
		return atts, nil
	}
	mc, err := newMaxCoverProblem(atts, len(atts))
	if err != nil {
		return atts, err
	}
	return mc.cover()
}

// maxCoverProblem defines Maximum k-Coverage problem.
type maxCoverProblem struct {
	k            int
	atts         []*ethpb.Attestation
	candidates   maxCoverCandidateList
	coverage     bitfield.Bitlist
	aggregated   []*maxCoverCandidate
	unaggregated []*maxCoverCandidate
}

// maxCoverCandidate represents a candidate set to be used in aggregation.
type maxCoverCandidate struct {
	bits      *bitfield.Bitlist
	signature *[]byte
	score     uint64
	processed bool
}

// newMaxCoverProblem returns initialized Maximum k-Coverage problem, with all necessary pre-tests done.
func newMaxCoverProblem(atts []*ethpb.Attestation, k int) (*maxCoverProblem, error) {
	if len(atts) == 0 {
		return nil, ErrInvalidAttestationCount
	}
	if len(atts) < k {
		k = len(atts)
	}

	// Assert that all attestations have the same bitlist length.
	for i := 1; i < len(atts); i++ {
		if atts[i-1].AggregationBits.Len() != atts[i].AggregationBits.Len() {
			return nil, ErrBitsDifferentLen
		}
	}

	mc := &maxCoverProblem{
		k:            k,
		atts:         atts,
		candidates:   maxCoverCandidateList{},
		coverage:     bitfield.NewBitlist(atts[0].AggregationBits.Len()),
		aggregated:   make([]*maxCoverCandidate, 0, k),
		unaggregated: make([]*maxCoverCandidate, 0, len(atts)),
	}
	for i := 0; i < len(atts); i++ {
		mc.candidates = append(mc.candidates, &maxCoverCandidate{
			bits:      &atts[i].AggregationBits,
			signature: &atts[i].Signature,
			score:     atts[i].AggregationBits.Count(),
		})
	}

	return mc, nil
}

type maxCoverCandidateList []*maxCoverCandidate

func (cl *maxCoverCandidateList) filter(covered bitfield.Bitlist) *maxCoverCandidateList {
	cur, end := 0, len(*cl)
	for cur < end {
		if (*cl)[cur].processed || covered.Overlaps(*(*cl)[cur].bits) {
			(*cl)[cur] = (*cl)[end-1]
			end--
			continue
		}
		cur++
	}
	*cl = (*cl)[:end]
	return cl
}

func (cl *maxCoverCandidateList) sort() *maxCoverCandidateList {
	sort.Slice(*cl, func(i, j int) bool {
		return (*cl)[i].score > (*cl)[j].score
	})
	return cl
}

func (mc *maxCoverProblem) cover() ([]*ethpb.Attestation, error) {
	if len(mc.candidates) == 0 {
		return mc.atts, nil
	}

	// Find coverage.
	for len(mc.aggregated) < mc.k && len(mc.candidates) > 0 {
		// Filter out processed and overlapping, sort by score in a descending order.
		mc.candidates.filter(mc.coverage).sort()

		// Pick enough non-overlapping candidates.
		for _, candidate := range mc.candidates {
			if candidate.processed {
				continue
			}
			if mc.coverage.Overlaps(*candidate.bits) {
				// Overlapping candidates violate non-intersection invariant.
				mc.unaggregated = append(mc.unaggregated, candidate)
				candidate.processed = true
			} else {
				mc.coverage = mc.coverage.Or(*candidate.bits)
				mc.aggregated = append(mc.aggregated, candidate)
				candidate.processed = true
			}
			if len(mc.aggregated) >= mc.k {
				break
			}
		}
	}

	// Remove processed items.
	mc.candidates.filter(mc.coverage)

	// Return results, start with the aggregated attestation.
	atts := make([]*ethpb.Attestation, 0, len(mc.aggregated)+len(mc.unaggregated))
	if len(mc.aggregated) > 0 {
		// Collect selected signatures.
		signs := make([]*bls.Signature, len(mc.aggregated))
		for i := 0; i < len(mc.aggregated); i++ {
			sig, err := signatureFromBytes(*mc.aggregated[i].signature)
			if err != nil {
				return nil, err
			}
			signs[i] = sig
		}
		atts = append(atts, &ethpb.Attestation{
			AggregationBits: mc.coverage,
			Data:            stateTrie.CopyAttestationData(mc.atts[0].Data),
			Signature:       aggregateSignatures(signs).Marshal(),
		})
	}

	// Add unaggregated attestations.
	for _, candidate := range mc.unaggregated {
		atts = append(atts, &ethpb.Attestation{
			AggregationBits: bytesutil.SafeCopyBytes(*candidate.bits),
			Data:            stateTrie.CopyAttestationData(mc.atts[0].Data),
			Signature:       bytesutil.SafeCopyBytes(*candidate.signature),
		})
	}

	// Add left-over candidates (their fate was undecided as we had k items already).
	for _, candidate := range mc.candidates {
		if candidate.processed {
			continue
		}
		atts = append(atts, &ethpb.Attestation{
			AggregationBits: bytesutil.SafeCopyBytes(*candidate.bits),
			Data:            stateTrie.CopyAttestationData(mc.atts[0].Data),
			Signature:       bytesutil.SafeCopyBytes(*candidate.signature),
		})
	}

	return atts, nil
}
