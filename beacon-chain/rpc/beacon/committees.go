package beacon

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListBeaconCommittees for a given epoch.
//
// This request may specify validator indices or public keys to filter out
// validator beacon chain committees. If no filter criteria is specified, the response returns
// all beacon committees for the current epoch. The results are paginated by default.
func (bs *Server) ListBeaconCommittees(
	ctx context.Context,
	req *ethpb.ListCommitteesRequest,
) (*ethpb.BeaconCommittees, error) {
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"requested page size %d can not be greater than max size %d",
			req.PageSize,
			params.BeaconConfig().MaxPageSize,
		)
	}

	headState := bs.HeadFetcher.HeadState()
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
	var err error
	if requestingGenesis || startSlot != headState.Slot {
		activeIndices, err = helpers.ActiveValidatorIndices(headState, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, err
		}
		archivedCommitteeInfo, err := bs.BeaconDB.ArchivedCommitteeInfo(ctx, helpers.SlotToEpoch(startSlot))
		if err != nil {
			return nil, err
		}
		attesterSeed = bytesutil.ToBytes32(archivedCommitteeInfo.AttesterSeed)
	} else {
		// Otherwise, we use data from the current epoch.
		currentEpoch := helpers.SlotToEpoch(headState.Slot)
		activeIndices, err = helpers.ActiveValidatorIndices(headState, currentEpoch)
		if err != nil {
			return nil, err
		}
		attesterSeed, err = helpers.Seed(headState, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
		if err != nil {
			return nil, err
		}
	}

	// If current epoch, compute. Otherwise, fetch from archive.
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
				return nil, err
			}
			committees = append(committees, &ethpb.BeaconCommittees_CommitteeItem{
				Committee: committee,
				Slot:      slot,
			})
		}
	}

	numCommittees := len(committees)
	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), numCommittees)
	if err != nil {
		return nil, err
	}
	activeIndices, err = helpers.ActiveValidatorIndices(headState, 0)
	if err != nil {
		return nil, err
	}
	fmt.Println(activeIndices)
	return &ethpb.BeaconCommittees{
		Epoch:                helpers.SlotToEpoch(startSlot),
		ActiveValidatorCount: uint64(len(activeIndices)),
		Committees:           committees[start:end],
		TotalSize:            int32(numCommittees),
		NextPageToken:        nextPageToken,
	}, nil
}
