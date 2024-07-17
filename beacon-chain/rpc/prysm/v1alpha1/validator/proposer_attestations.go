package validator

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation"
	attaggregation "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1/attestation/aggregation/attestations"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

type proposerAtts []ethpb.Att

func (vs *Server) packAttestations(ctx context.Context, latestState state.BeaconState, blkSlot primitives.Slot) ([]ethpb.Att, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.packAttestations")
	defer span.End()

	atts := vs.AttPool.AggregatedAttestations()
	atts, err := vs.validateAndDeleteAttsInPool(ctx, latestState, atts)
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
	atts = append(atts, uAtts...)

	// Checking the state's version here will give the wrong result if the last slot of Deneb is missed.
	// The head state will still be in Deneb while we are trying to build an Electra block.
	postElectra := slots.ToEpoch(blkSlot) >= params.BeaconConfig().ElectraForkEpoch

	versionAtts := make([]ethpb.Att, 0, len(atts))
	if postElectra {
		for _, a := range atts {
			if a.Version() == version.Electra {
				versionAtts = append(versionAtts, a)
			}
		}
	} else {
		for _, a := range atts {
			if a.Version() == version.Phase0 {
				versionAtts = append(versionAtts, a)
			}
		}
	}

	// Remove duplicates from both aggregated/unaggregated attestations. This
	// prevents inefficient aggregates being created.
	versionAtts, err = proposerAtts(versionAtts).dedup()
	if err != nil {
		return nil, err
	}

	attsById := make(map[attestation.Id][]ethpb.Att, len(versionAtts))
	for _, att := range versionAtts {
		id, err := attestation.NewId(att, attestation.Data)
		if err != nil {
			return nil, errors.Wrap(err, "could not create attestation ID")
		}
		attsById[id] = append(attsById[id], att)
	}

	for id, as := range attsById {
		as, err := attaggregation.Aggregate(as)
		if err != nil {
			return nil, err
		}
		attsById[id] = as
	}

	var attsForInclusion proposerAtts
	if postElectra {
		// TODO: hack for Electra devnet-1, take only one aggregate per ID
		// (which essentially means one aggregate for an attestation_data+committee combination
		topAggregates := make([]ethpb.Att, 0)
		for _, v := range attsById {
			topAggregates = append(topAggregates, v[0])
		}

		attsForInclusion, err = computeOnChainAggregate(topAggregates)
		if err != nil {
			return nil, err
		}
	} else {
		attsForInclusion = make([]ethpb.Att, 0)
		for _, as := range attsById {
			attsForInclusion = append(attsForInclusion, as...)
		}
	}

	deduped, err := attsForInclusion.dedup()
	if err != nil {
		return nil, err
	}
	sorted, err := deduped.sortByProfitability()
	if err != nil {
		return nil, err
	}
	atts = sorted.limitToMaxAttestations()

	atts, err = vs.filterAttestationBySignature(ctx, atts, latestState)
	if err != nil {
		return nil, err
	}

	return atts, nil
}

// filter separates attestation list into two groups: valid and invalid attestations.
// The first group passes the all the required checks for attestation to be considered for proposing.
// And attestations from the second group should be deleted.
func (a proposerAtts) filter(ctx context.Context, st state.BeaconState) (proposerAtts, proposerAtts) {
	validAtts := make([]ethpb.Att, 0, len(a))
	invalidAtts := make([]ethpb.Att, 0, len(a))

	for _, att := range a {
		if err := blocks.VerifyAttestationNoVerifySignature(ctx, st, att); err == nil {
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
	if len(a) == 0 {
		return a
	}

	var limit uint64
	if a[0].Version() == version.Phase0 {
		limit = params.BeaconConfig().MaxAttestations
	} else {
		limit = params.BeaconConfig().MaxAttestationsElectra
	}
	if uint64(len(a)) > limit {
		return a[:limit]
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
	attsByDataRoot := make(map[attestation.Id][]ethpb.Att, len(a))
	for _, att := range a {
		id, err := attestation.NewId(att, attestation.Data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create attestation ID")
		}
		attsByDataRoot[id] = append(attsByDataRoot[id], att)
	}

	uniqAtts := make([]ethpb.Att, 0, len(a))
	for _, atts := range attsByDataRoot {
		for i := 0; i < len(atts); i++ {
			a := atts[i]
			for j := i + 1; j < len(atts); j++ {
				b := atts[j]
				if c, err := a.GetAggregationBits().Contains(b.GetAggregationBits()); err != nil {
					return nil, err
				} else if c {
					// a contains b, b is redundant.
					atts[j] = atts[len(atts)-1]
					atts[len(atts)-1] = nil
					atts = atts[:len(atts)-1]
					j--
				} else if c, err := b.GetAggregationBits().Contains(a.GetAggregationBits()); err != nil {
					return nil, err
				} else if c {
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

	return uniqAtts, nil
}

// This filters the input attestations to return a list of valid attestations to be packaged inside a beacon block.
func (vs *Server) validateAndDeleteAttsInPool(ctx context.Context, st state.BeaconState, atts []ethpb.Att) ([]ethpb.Att, error) {
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
func (vs *Server) deleteAttsInPool(ctx context.Context, atts []ethpb.Att) error {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.deleteAttsInPool")
	defer span.End()

	for _, att := range atts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if helpers.IsAggregated(att) {
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

// isAttestationFromCurrentEpoch returns true if the attestation is from the current epoch.
func (vs *Server) isAttestationFromCurrentEpoch(att ethpb.Att) bool {
	currentSlot := vs.TimeFetcher.CurrentSlot()
	currentEpoch := slots.ToEpoch(currentSlot)
	attestationSlot := att.GetData().Slot
	attestationEpoch := slots.ToEpoch(attestationSlot)
	return attestationEpoch == currentEpoch
}

// isAttestationFromPreviousEpoch returns true if the attestation is from the previous epoch.
func (vs *Server) isAttestationFromPreviousEpoch(att ethpb.Att) bool {
	currentSlot := vs.TimeFetcher.CurrentSlot()
	previousEpoch := slots.ToEpoch(currentSlot)
	if previousEpoch >= 1 {
		previousEpoch = previousEpoch.Sub(1)
	}
	attestationSlot := att.GetData().Slot
	attestationEpoch := slots.ToEpoch(attestationSlot)
	return attestationEpoch == previousEpoch
}

// filterCurrentEpochAttestationByForkchoice filters attestations from the current epoch based on fork choice conditions.
// Returns true if any of the following conditions are met:
// 1. The attestation beacon block root is for a slot in the previous epoch (according to fork choice).
// 2. The attestation target root is the same as the attestation beacon block root.
// 3. The common ancestor of the head block and the attestation beacon block root is from the previous epoch.
func (vs *Server) filterCurrentEpochAttestationByForkchoice(ctx context.Context, att ethpb.Att, headRoot [32]byte) (bool, error) {
	if !vs.isAttestationFromCurrentEpoch(att) {
		return false, nil
	}

	attTargetRoot := [32]byte(att.GetData().Target.Root)
	attBlockRoot := [32]byte(att.GetData().BeaconBlockRoot)
	if attBlockRoot == attTargetRoot {
		return true, nil
	}

	slot, err := vs.ForkchoiceFetcher.RecentBlockSlot(attBlockRoot)
	if err != nil {
		return false, err
	}
	epoch := slots.ToEpoch(slot)
	prevEpoch := slots.ToEpoch(vs.TimeFetcher.CurrentSlot())
	if prevEpoch >= 1 {
		prevEpoch = prevEpoch.Sub(1)
	}
	if epoch == prevEpoch {
		return true, nil
	}

	_, slot, err = vs.ForkchoiceFetcher.CommonAncestor(ctx, headRoot, attBlockRoot)
	if err != nil {
		return false, err
	}
	epoch = slots.ToEpoch(slot)
	return epoch == prevEpoch, nil
}

// filterCurrentEpochAttestationByTarget returns true if an attestation from the current epoch matches the fork choice target view.
// The conditions checked are:
// 1. The attestation's target epoch matches the forkchoice target epoch.
// 2. The attestation's target root matches the forkchoice target root.
func (vs *Server) filterCurrentEpochAttestationByTarget(att ethpb.Att, targetRoot [32]byte, targetEpoch primitives.Epoch) (bool, error) {
	if !vs.isAttestationFromCurrentEpoch(att) {
		return false, nil
	}

	attTargetRoot := [32]byte(att.GetData().Target.Root)
	return att.GetData().Target.Epoch == targetEpoch && attTargetRoot == targetRoot, nil
}

// filterPreviousEpochAttestationByTarget returns true if an attestation from the previous epoch matches the fork choice previous target view.
// The conditions checked are:
// 1. The attestation's target epoch matches the forkchoice previous target epoch.
// 2. The attestation's target root matches the forkchoice previous target root.
func (vs *Server) filterPreviousEpochAttestationByTarget(att ethpb.Att, targetRoot [32]byte, targetEpoch primitives.Epoch) (bool, error) {
	if !vs.isAttestationFromPreviousEpoch(att) {
		return false, nil
	}

	attTargetRoot := [32]byte(att.GetData().Target.Root)
	return att.GetData().Target.Epoch == targetEpoch && attTargetRoot == targetRoot, nil
}

// filterAttestationBySignature filters attestations based on specific conditions and performs batch signature verification.
// The conditions checked are:
// 1. The attestation matches the current target view defined in `filterCurrentEpochAttestationByTarget`.
// 2. The attestation matches the previous target view defined in `filterPreviousEpochAttestationByTarget`.
// 3. The attestation matches certain fork choice conditions defined in `filterCurrentEpochAttestationByForkchoice`.
// The remaining attestations are sent for batch signature verification. If the batch verification fails, each signature is verified individually.
func (vs *Server) filterAttestationBySignature(ctx context.Context, atts proposerAtts, st state.BeaconState) (proposerAtts, error) {
	headSlot := vs.HeadFetcher.HeadSlot()
	targetEpoch := slots.ToEpoch(headSlot)
	r, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, err
	}
	headRoot := [32]byte(r)

	targetRoot, err := vs.HeadFetcher.TargetRootForEpoch(headRoot, targetEpoch)
	if err != nil {
		return nil, err
	}

	prevTargetEpoch := primitives.Epoch(0)
	if targetEpoch >= 1 {
		prevTargetEpoch = targetEpoch.Sub(1)
	}
	prevTargetRoot, err := vs.HeadFetcher.TargetRootForEpoch(headRoot, prevTargetEpoch)
	if err != nil {
		return nil, err
	}

	var verifiedAtts proposerAtts
	var unverifiedAtts proposerAtts
	for _, att := range atts {
		ok, err := vs.filterCurrentEpochAttestationByTarget(att, targetRoot, targetEpoch)
		if err != nil {
			log.WithFields(attestationFields(att)).WithError(err).Error("Could not filter current epoch attestation by target")
		}
		if ok {
			verifiedAtts = append(verifiedAtts, att)
			continue
		}

		ok, err = vs.filterPreviousEpochAttestationByTarget(att, prevTargetRoot, prevTargetEpoch)
		if err != nil {
			log.WithFields(attestationFields(att)).WithError(err).Error("Could not filter previous epoch attestation by target")
		}
		if ok {
			verifiedAtts = append(verifiedAtts, att)
			continue
		}

		ok, err = vs.filterCurrentEpochAttestationByForkchoice(ctx, att, headRoot)
		if err != nil {
			log.WithFields(attestationFields(att)).WithError(err).Error("Could not filter current epoch attestation by fork choice")
		}
		if ok {
			verifiedAtts = append(verifiedAtts, att)
			continue
		}

		unverifiedAtts = append(unverifiedAtts, att)
	}

	if len(unverifiedAtts) == 0 {
		return verifiedAtts, nil
	}

	unverifiedAtts = unverifiedAtts.filterBatchSignature(ctx, st)

	return append(verifiedAtts, unverifiedAtts...), nil
}

// filterBatchSignature verifies the signatures of the attestation set.
// If batch verification fails, the attestation set is filtered by verifying each signature individually.
func (a proposerAtts) filterBatchSignature(ctx context.Context, st state.BeaconState) proposerAtts {
	aSet, err := blocks.AttestationSignatureBatch(ctx, st, a)
	if err != nil {
		log.WithError(err).Error("Could not create attestation signature set")
		return a.filterIndividualSignature(ctx, st)
	}

	if verified, err := aSet.Verify(); err != nil || !verified {
		if err != nil {
			log.WithError(err).Error("Batch verification failed")
		} else {
			log.Error("Batch verification failed: signatures not verified")
		}
		return a.filterIndividualSignature(ctx, st)
	}
	return a
}

// filterIndividualSignature filters the attestation set by verifying each signature individually.
func (a proposerAtts) filterIndividualSignature(ctx context.Context, st state.BeaconState) proposerAtts {
	var validAtts proposerAtts
	for _, att := range a {
		aSet, err := blocks.AttestationSignatureBatch(ctx, st, []ethpb.Att{att})
		if err != nil {
			log.WithFields(attestationFields(att)).WithError(err).Error("Could not create individual attestation signature set")
			continue
		}
		if verified, err := aSet.Verify(); err != nil || !verified {
			logEntry := log.WithFields(attestationFields(att))
			if err != nil {
				logEntry.WithError(err).Error("Verification of individual attestation failed")
			} else {
				logEntry.Error("Verification of individual attestation failed: signature not verified")
			}
			continue
		}
		validAtts = append(validAtts, att)
	}
	return validAtts
}

func attestationFields(att ethpb.Att) logrus.Fields {
	return logrus.Fields{
		"slot":            att.GetData().Slot,
		"index":           att.GetData().CommitteeIndex,
		"targetRoot":      fmt.Sprintf("%x", att.GetData().Target.Root),
		"targetEpoch":     att.GetData().Target.Epoch,
		"beaconBlockRoot": fmt.Sprintf("%x", att.GetData().BeaconBlockRoot),
	}
}
