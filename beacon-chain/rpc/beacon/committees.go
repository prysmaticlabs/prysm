package beacon

import (
	"context"
	"strconv"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
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
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			params.BeaconConfig().MaxPageSize,
		)
	}

	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}

	var requestingGenesis bool
	var startSlot uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListCommitteesRequest_Epoch:
		startSlot = helpers.StartSlot(q.Epoch)
	case *ethpb.ListCommitteesRequest_Genesis:
		requestingGenesis = q.Genesis
	default:
		startSlot = headState.Slot
	}

	var attesterSeed [32]byte
	var activeIndices []uint64
	// This is the archival condition, if the requested epoch is < current epoch or if we are
	// requesting data from the genesis epoch.
	if requestingGenesis || helpers.SlotToEpoch(startSlot) < helpers.SlotToEpoch(headState.Slot) {
		activeIndices, err = helpers.ActiveValidatorIndices(headState, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		archivedCommitteeInfo, err := bs.BeaconDB.ArchivedCommitteeInfo(ctx, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not request archival data for epoch %d: %v",
				helpers.SlotToEpoch(startSlot),
				err,
			)
		}
		if archivedCommitteeInfo == nil {
			return nil, status.Errorf(
				codes.NotFound,
				"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
				helpers.SlotToEpoch(startSlot),
			)
		}
		attesterSeed = bytesutil.ToBytes32(archivedCommitteeInfo.AttesterSeed)
	} else if !requestingGenesis && helpers.SlotToEpoch(startSlot) == helpers.SlotToEpoch(headState.Slot) {
		// Otherwise, we use data from the current epoch.
		currentEpoch := helpers.SlotToEpoch(headState.Slot)
		activeIndices, err = helpers.ActiveValidatorIndices(headState, currentEpoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for current epoch %d: %v",
				currentEpoch,
				err,
			)
		}
		attesterSeed, err = helpers.Seed(headState, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve attester seed for current epoch %d: %v",
				currentEpoch,
				err,
			)
		}
	} else {
		// Otherwise, we are requesting data from the future and we return an error.
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			helpers.CurrentEpoch(headState),
			helpers.SlotToEpoch(startSlot),
		)
	}

	committees := make([]*ethpb.BeaconCommittees_CommitteeItem, 0)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		for i := uint64(0); i < countAtSlot; i++ {
			epochOffset := i + (slot%params.BeaconConfig().SlotsPerEpoch)*countAtSlot
			totalCount := countAtSlot * params.BeaconConfig().SlotsPerEpoch
			committee, err := helpers.ComputeCommittee(activeIndices, attesterSeed, epochOffset, totalCount)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"Could not compute committee for slot %d: %v",
					slot,
					err,
				)
			}
			committees = append(committees, &ethpb.BeaconCommittees_CommitteeItem{
				Committee: committee,
				Slot:      slot,
			})
		}
	}

	numCommittees := len(committees)
	// If there are no committees, we simply return a response specifying this.
	// Otherwise, attempting to paginate 0 committees below would result in an error.
	if numCommittees == 0 {
		return &ethpb.BeaconCommittees{
			Epoch:                helpers.SlotToEpoch(startSlot),
			ActiveValidatorCount: uint64(len(activeIndices)),
			Committees:           make([]*ethpb.BeaconCommittees_CommitteeItem, 0),
			TotalSize:            int32(0),
			NextPageToken:        strconv.Itoa(0),
		}, nil
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numCommittees)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Could not paginate results: %v",
			err,
		)
	}
	return &ethpb.BeaconCommittees{
		Epoch:                helpers.SlotToEpoch(startSlot),
		ActiveValidatorCount: uint64(len(activeIndices)),
		Committees:           committees[start:end],
		TotalSize:            int32(numCommittees),
		NextPageToken:        nextPageToken,
	}, nil
}
