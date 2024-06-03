package validator

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation"
	attaggregation "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"go.opencensus.io/trace"
)

type proposerAtts []consensusblocks.ROAttestation

func (vs *Server) packAttestations(ctx context.Context, latestState state.BeaconState) ([]ethpb.Att, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.packAttestations")
	defer span.End()

	roAtts := vs.AttPool.AggregatedAttestations()
	roAtts, err := vs.validateAndDeleteAttsInPool(ctx, latestState, roAtts)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter attestations")
	}

	uAtts, err := vs.AttPool.UnaggregatedAttestations()
	if err != nil {
		return nil, errors.Wrap(err, "could not get unaggregated attestations")
	}
	uAtts, err = vs.validateAndDeleteAttsInPool(ctx, latestState, uAtts)
	if err != nil {
		return nil, errors.Wrap(err, "could not filter attestations")
	}
	roAtts = append(roAtts, uAtts...)

	// Remove duplicates from both aggregated/unaggregated attestations. This
	// prevents inefficient aggregates being created.
	roAtts, err = proposerAtts(roAtts).dedup()
	if err != nil {
		return nil, err
	}

	attsByDataRoot := make(map[consensusblocks.AttestationId][]consensusblocks.ROAttestation, len(roAtts))
	for _, att := range roAtts {
		attsByDataRoot[att.DataId()] = append(attsByDataRoot[att.DataId()], att)
	}

	for r, as := range attsByDataRoot {
		as, err := attaggregation.Aggregate(as)
		if err != nil {
			return nil, err
		}
		attsByDataRoot[r] = as
	}

	attsForInclusion := proposerAtts(make([]consensusblocks.ROAttestation, 0))
	for _, as := range attsByDataRoot {
		attsForInclusion = append(attsForInclusion, as...)
	}

	deduped, err := attsForInclusion.dedup()
	if err != nil {
		return nil, err
	}
	sorted, err := deduped.sortByProfitability()
	if err != nil {
		return nil, err
	}
	roAtts = sorted.limitToMaxAttestations()

	atts := make([]ethpb.Att, 0, len(roAtts))
	for i, att := range roAtts {
		atts[i] = att.Att
	}
	return atts, nil
}

// filter separates attestation list into two groups: valid and invalid attestations.
// The first group passes the all the required checks for attestation to be considered for proposing.
// And attestations from the second group should be deleted.
func (a proposerAtts) filter(ctx context.Context, st state.BeaconState) (proposerAtts, proposerAtts) {
	validAtts := make([]consensusblocks.ROAttestation, 0, len(a))
	invalidAtts := make([]consensusblocks.ROAttestation, 0, len(a))

	for _, att := range a {
		if err := blocks.VerifyAttestationNoVerifySignature(ctx, st, att.Att); err == nil {
			validAtts = append(validAtts, att)
			continue
		}
		invalidAtts = append(invalidAtts, att)
	}
	return validAtts, invalidAtts
}

// sortByProfitability orders attestations by highest slot and by highest aggregation bit count.
func (a proposerAtts) sortByProfitability() (proposerAtts, error) {
	if len(a) < 2 {
		return a, nil
	}
	return a.sortByProfitabilityUsingMaxCover()
}

// sortByProfitabilityUsingMaxCover orders attestations by highest slot and by highest aggregation bit count.
// Duplicate bits are counted only once, using max-cover algorithm.
func (a proposerAtts) sortByProfitabilityUsingMaxCover() (proposerAtts, error) {
	// Separate attestations by slot, as slot number takes higher precedence when sorting.
	var slots []primitives.Slot
	attsBySlot := map[primitives.Slot]proposerAtts{}
	for _, att := range a {
		if _, ok := attsBySlot[att.GetData().Slot]; !ok {
			slots = append(slots, att.GetData().Slot)
		}
		attsBySlot[att.GetData().Slot] = append(attsBySlot[att.GetData().Slot], att)
	}

	selectAtts := func(atts proposerAtts) (proposerAtts, error) {
		if len(atts) < 2 {
			return atts, nil
		}
		candidates := make([]*bitfield.Bitlist64, len(atts))
		for i := 0; i < len(atts); i++ {
			var err error
			candidates[i], err = atts[i].GetAggregationBits().ToBitlist64()
			if err != nil {
				return nil, err
			}
		}
		// Add selected candidates on top, those that are not selected - append at bottom.
		selectedKeys, _, err := aggregation.MaxCover(candidates, len(candidates), true /* allowOverlaps */)
		if err == nil {
			// Pick selected attestations first, leftover attestations will be appended at the end.
			// Both lists will be sorted by number of bits set.
			selectedAtts := make(proposerAtts, selectedKeys.Count())
			leftoverAtts := make(proposerAtts, selectedKeys.Not().Count())
			for i, key := range selectedKeys.BitIndices() {
				selectedAtts[i] = atts[key]
			}
			for i, key := range selectedKeys.Not().BitIndices() {
				leftoverAtts[i] = atts[key]
			}
			sort.Slice(selectedAtts, func(i, j int) bool {
				return selectedAtts[i].GetAggregationBits().Count() > selectedAtts[j].GetAggregationBits().Count()
			})
			sort.Slice(leftoverAtts, func(i, j int) bool {
				return leftoverAtts[i].GetAggregationBits().Count() > leftoverAtts[j].GetAggregationBits().Count()
			})
			return append(selectedAtts, leftoverAtts...), nil
		}
		return atts, nil
	}

	// Select attestations. Slots are sorted from higher to lower at this point. Within slots attestations
	// are sorted to maximize profitability (greedily selected, with previous attestations' bits
	// evaluated before including any new attestation).
	var sortedAtts proposerAtts
	sort.Slice(slots, func(i, j int) bool {
		return slots[i] > slots[j]
	})
	for _, slot := range slots {
		selected, err := selectAtts(attsBySlot[slot])
		if err != nil {
			return nil, err
		}
		sortedAtts = append(sortedAtts, selected...)
	}

	return sortedAtts, nil
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
func (a proposerAtts) dedup() (proposerAtts, error) {
	if len(a) < 2 {
		return a, nil
	}
	attsByDataId := make(map[consensusblocks.AttestationId][]consensusblocks.ROAttestation, len(a))
	for _, att := range a {
		attsByDataId[att.DataId()] = append(attsByDataId[att.DataId()], att)
	}

	uniqAtts := make([]consensusblocks.ROAttestation, 0, len(a))
	for _, atts := range attsByDataId {
		for i := 0; i < len(atts); i++ {
			a := atts[i]
			for j := i + 1; j < len(atts); j++ {
				b := atts[j]
				if c, err := a.GetAggregationBits().Contains(b.GetAggregationBits()); err != nil {
					return nil, err
				} else if c {
					// a contains b, b is redundant.
					atts[j] = atts[len(atts)-1]
					atts[len(atts)-1] = consensusblocks.ROAttestation{}
					atts = atts[:len(atts)-1]
					j--
				} else if c, err := b.GetAggregationBits().Contains(a.GetAggregationBits()); err != nil {
					return nil, err
				} else if c {
					// b contains a, a is redundant.
					atts[i] = atts[len(atts)-1]
					atts[len(atts)-1] = consensusblocks.ROAttestation{}
					atts = atts[:len(atts)-1]
					i--
					break
				}
			}
		}
		uniqAtts = append(uniqAtts, atts...)
	}

	return uniqAtts, nil
}

// This filters the input attestations to return a list of valid attestations to be packaged inside a beacon block.
func (vs *Server) validateAndDeleteAttsInPool(
	ctx context.Context,
	st state.BeaconState,
	atts []consensusblocks.ROAttestation,
) ([]consensusblocks.ROAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.validateAndDeleteAttsInPool")
	defer span.End()

	validAtts, invalidAtts := proposerAtts(atts).filter(ctx, st)
	if err := vs.deleteAttsInPool(ctx, invalidAtts); err != nil {
		return nil, err
	}
	return validAtts, nil
}

// The input attestations are processed and seen by the node, this deletes them from pool
// so proposers don't include them in a block for the future.
func (vs *Server) deleteAttsInPool(ctx context.Context, atts []consensusblocks.ROAttestation) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.deleteAttsInPool")
	defer span.End()

	for _, att := range atts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if helpers.IsAggregated(att.Att) {
			if err := vs.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := vs.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}
