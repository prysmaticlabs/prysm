package validator

import (
	"context"
	"sort"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type proposerAtts []*ethpb.Attestation

// filter separates attestation list into two groups: valid and invalid attestations.
// The first group passes the all the required checks for attestation to be considered for proposing.
// And attestations from the second group should be deleted.
func (al proposerAtts) filter(ctx context.Context, state *stateTrie.BeaconState) (proposerAtts, proposerAtts) {
	validAtts := make([]*ethpb.Attestation, 0, len(al))
	invalidAtts := make([]*ethpb.Attestation, 0, len(al))
	for _, att := range al {
		if _, err := blocks.ProcessAttestation(ctx, state, att); err == nil {
			validAtts = append(validAtts, att)
			continue
		}
		invalidAtts = append(invalidAtts, att)
	}
	return validAtts, invalidAtts
}

// sortByProfitability orders attestations by highest slot and by highest aggregation bit count.
func (al proposerAtts) sortByProfitability() proposerAtts {
	if len(al) < 2 {
		return al
	}
	sort.Slice(al, func(i, j int) bool {
		if al[i].Data.Slot == al[j].Data.Slot {
			return al[i].AggregationBits.Count() > al[j].AggregationBits.Count()
		}
		return al[i].Data.Slot > al[j].Data.Slot
	})
	return al
}

// limitToMaxAttestations limits attestations to maximum attestations per block.
func (al proposerAtts) limitToMaxAttestations() proposerAtts {
	if uint64(len(al)) > params.BeaconConfig().MaxAttestations {
		return al[:params.BeaconConfig().MaxAttestations]
	}
	return al
}

// dedup removes duplicate attestations (ones with the same bits set on).
// Important: not only exact duplicates are removed, but proper subsets are removed too
// (their known bits are redundant and are already contained in their supersets).
func (al proposerAtts) dedup() proposerAtts {
	if len(al) < 2 {
		return al
	}
	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(al))
	for _, att := range al {
		attDataRoot, err := att.Data.HashTreeRoot()
		if err != nil {
			continue
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
	}

	uniqAtts := make([]*ethpb.Attestation, 0, len(al))
	for _, atts := range attsByDataRoot {
		for i := 0; i < len(atts); i++ {
			a := atts[i]
			for j := i + 1; j < len(atts); j++ {
				b := atts[j]
				if a.AggregationBits.Contains(b.AggregationBits) {
					// a contains b, b is redundant.
					atts[j] = atts[len(atts)-1]
					atts[len(atts)-1] = nil
					atts = atts[:len(atts)-1]
					j--
				} else if b.AggregationBits.Contains(a.AggregationBits) {
					// b contains a, a is redundant.
					atts[i] = atts[len(atts)-1]
					atts[len(atts)-1] = nil
					atts = atts[:len(atts)-1]
					i--
					break
				}
			}
		}
		uniqAtts = append(uniqAtts, atts...)
	}

	return uniqAtts
}
