/*
// TODO(3603): Delete this file when issue 3603 closes.
// When there's a signature fails to verify with committee cache enabled at run time,
// this files defines all the helpers to rerun signature verify routine without cache in play.
// This provides extra assurance that committee cache can't break run time.
*/

package forkchoice

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
)

// convertToIndexed converts attestation to indexed form without the usage of cache even when cache flag is enabled.
func convertToIndexed(ctx context.Context, state *pb.BeaconState, attestation *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.ConvertToIndexed")
	defer span.End()

	attIndices, err := attestingIndices(state, attestation.Data, attestation.AggregationBits)
	if err != nil {
		return nil, errors.Wrap(err, "could not get attesting indices")
	}

	cb1i, err := attestingIndices(state, attestation.Data, attestation.CustodyBits)
	if err != nil {
		return nil, err
	}
	if !sliceutil.SubsetUint64(cb1i, attIndices) {
		return nil, fmt.Errorf("%v is not a subset of %v", cb1i, attIndices)
	}
	cb1Map := make(map[uint64]bool)
	for _, idx := range cb1i {
		cb1Map[idx] = true
	}
	cb0i := []uint64{}
	for _, idx := range attIndices {
		if !cb1Map[idx] {
			cb0i = append(cb0i, idx)
		}
	}
	sort.Slice(cb0i, func(i, j int) bool {
		return cb0i[i] < cb0i[j]
	})

	sort.Slice(cb1i, func(i, j int) bool {
		return cb1i[i] < cb1i[j]
	})
	inAtt := &ethpb.IndexedAttestation{
		Data:                attestation.Data,
		Signature:           attestation.Signature,
		CustodyBit_0Indices: cb0i,
		CustodyBit_1Indices: cb1i,
	}
	return inAtt, nil
}

// attestingIndices gets attesting validator indices from the attested data and bitfield.
func attestingIndices(state *pb.BeaconState, data *ethpb.AttestationData, bf bitfield.Bitfield) ([]uint64, error) {
	committee, err := beaconCommittee(state, data.Slot, data.CommitteeIndex)
	if err != nil {
		return nil, errors.Wrap(err, "could not get committee")
	}

	indices := make([]uint64, 0, len(committee))
	indicesSet := make(map[uint64]bool)
	for i, idx := range committee {
		if !indicesSet[idx] {
			if bf.BitAt(uint64(i)) {
				indices = append(indices, idx)
			}
		}
		indicesSet[idx] = true
	}
	return indices, nil
}

// beaconCommittee gets the commmittee of a given slot and committee index.
func beaconCommittee(state *pb.BeaconState, slot uint64, committeeIndex uint64) ([]uint64, error) {
	epoch := helpers.SlotToEpoch(slot)

	committeesPerSlot, err := helpers.CommitteeCountAtSlot(state, slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not get committee count at slot")
	}
	epochOffset := committeeIndex + (slot%params.BeaconConfig().SlotsPerEpoch)*committeesPerSlot
	count := committeesPerSlot * params.BeaconConfig().SlotsPerEpoch

	seed, err := helpers.Seed(state, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, errors.Wrap(err, "could not get seed")
	}

	indices, err := helpers.ActiveValidatorIndices(state, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not get active indices")
	}

	return helpers.ComputeCommittee(indices, seed, epochOffset, count)
}
