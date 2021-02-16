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
func (a proposerAtts) filter(ctx context.Context, state *stateTrie.BeaconState) (proposerAtts, proposerAtts) {
	validAtts := make([]*ethpb.Attestation, 0, len(a))
	invalidAtts := make([]*ethpb.Attestation, 0, len(a))
	for _, att := range a {
		if _, err := blocks.ProcessAttestationNoVerifySignature(ctx, state, att); err == nil {
			validAtts = append(validAtts, att)
			continue
		}
		invalidAtts = append(invalidAtts, att)
	}
	return validAtts, invalidAtts
}

// sortByProfitability orders attestations by highest slot and by highest aggregation bit count.
func (a proposerAtts) sortByProfitability() proposerAtts {
	if len(a) < 2 {
		return a
	}
	sort.Slice(a, func(i, j int) bool {
		if a[i].Data.Slot == a[j].Data.Slot {
			return a[i].AggregationBits.Count() > a[j].AggregationBits.Count()
		}
		return a[i].Data.Slot > a[j].Data.Slot
	})
	return a
}

// limitToMaxAttestations limits attestations to maximum attestations per block.
func (a proposerAtts) limitToMaxAttestations() proposerAtts {
	if uint64(len(a)) > params.BeaconConfig().MaxAttestations {
		return a[:params.BeaconConfig().MaxAttestations]
	}
	return a
}

// dedup removes duplicate attestations (ones with the same bits set on).
// Important: not only exact duplicates are removed, but proper subsets are removed too
// (their known bits are redundant and are already contained in their supersets).
func (a proposerAtts) dedup() proposerAtts {
	if len(a) < 2 {
		return a
	}
	attsByDataRoot := make(map[[32]byte][]*ethpb.Attestation, len(a))
	for _, att := range a {
		attDataRoot, err := att.Data.HashTreeRoot()
		if err != nil {
			continue
		}
		attsByDataRoot[attDataRoot] = append(attsByDataRoot[attDataRoot], att)
	}

	uniqAtts := make([]*ethpb.Attestation, 0, len(a))
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
