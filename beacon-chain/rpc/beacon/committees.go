package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListBeaconCommittees for a given epoch.
//
// If no filter criteria is specified, the response returns
// all beacon committees for the current epoch. The results are paginated by default.
func (bs *Server) ListBeaconCommittees(
	ctx context.Context,
	req *ethpb.ListCommitteesRequest,
) (*ethpb.BeaconCommittees, error) {

	var requestingGenesis bool
	var startSlot uint64
	headSlot := bs.HeadFetcher.HeadSlot()
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListCommitteesRequest_Epoch:
		startSlot = helpers.StartSlot(q.Epoch)
	case *ethpb.ListCommitteesRequest_Genesis:
		requestingGenesis = q.Genesis
		if !requestingGenesis {
			startSlot = headSlot
		}
	default:
		startSlot = headSlot
	}
	committees, activeIndices, err := bs.retrieveCommitteesForEpoch(ctx, helpers.SlotToEpoch(startSlot))
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not retrieve committees for epoch %d: %v",
			helpers.SlotToEpoch(startSlot),
			err,
		)
	}
	return &ethpb.BeaconCommittees{
		Epoch:                helpers.SlotToEpoch(startSlot),
		Committees:           committees,
		ActiveValidatorCount: uint64(len(activeIndices)),
	}, nil
}

func (bs *Server) retrieveCommitteesForEpoch(
	ctx context.Context,
	epoch uint64,
) (map[uint64]*ethpb.BeaconCommittees_CommitteesList, []uint64, error) {
	var attesterSeed [32]byte
	var activeIndices []uint64
	var err error
	startSlot := helpers.StartSlot(epoch)
	headEpoch := helpers.SlotToEpoch(bs.HeadFetcher.HeadSlot())
	if helpers.SlotToEpoch(startSlot)+1 < headEpoch {
		activeIndices, err = bs.HeadFetcher.HeadValidatorsIndices(helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		archivedCommitteeInfo, err := bs.BeaconDB.ArchivedCommitteeInfo(ctx, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Could not request archival data for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		if archivedCommitteeInfo == nil {
			return nil, nil, status.Errorf(
				codes.NotFound,
				"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
				helpers.SlotToEpoch(startSlot),
			)
		}
		attesterSeed = bytesutil.ToBytes32(archivedCommitteeInfo.AttesterSeed)
	} else if helpers.SlotToEpoch(startSlot)+1 == headEpoch || helpers.SlotToEpoch(startSlot) == headEpoch {
		// Otherwise, we use current beacon state to calculate the committees.
		requestedEpoch := helpers.SlotToEpoch(startSlot)
		activeIndices, err = bs.HeadFetcher.HeadValidatorsIndices(requestedEpoch)
		if err != nil {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for requested epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
		attesterSeed, err = bs.HeadFetcher.HeadSeed(requestedEpoch)
		if err != nil {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Could not retrieve attester seed for requested epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
	} else {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			headEpoch,
			helpers.SlotToEpoch(startSlot),
		)
	}

	committeesListsBySlot, err := computeCommittees(startSlot, activeIndices, attesterSeed)
	if err != nil {
		return nil, nil, status.Errorf(
			codes.InvalidArgument,
			"Could not compute committees for epoch %d: %v",
			helpers.SlotToEpoch(startSlot),
			err,
		)
	}
	return committeesListsBySlot, activeIndices, nil
}

// Compute committees given a start slot, active validator indices, and
// the attester seeds value.
func computeCommittees(
	startSlot uint64,
	activeIndices []uint64,
	attesterSeed [32]byte,
) (map[uint64]*ethpb.BeaconCommittees_CommitteesList, error) {
	committeesListsBySlot := make(map[uint64]*ethpb.BeaconCommittees_CommitteesList)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		committeeItems := make([]*ethpb.BeaconCommittees_CommitteeItem, countAtSlot)
		for committeeIndex := uint64(0); committeeIndex < countAtSlot; committeeIndex++ {
			committee, err := helpers.BeaconCommittee(activeIndices, attesterSeed, slot, committeeIndex)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"Could not compute committee for slot %d: %v",
					slot,
					err,
				)
			}
			committeeItems[committeeIndex] = &ethpb.BeaconCommittees_CommitteeItem{
				ValidatorIndices: committee,
			}
		}
		committeesListsBySlot[slot] = &ethpb.BeaconCommittees_CommitteesList{
			Committees: committeeItems,
		}
	}
	return committeesListsBySlot, nil
}
