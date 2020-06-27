package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
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

	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	var requestedSlot uint64
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListCommitteesRequest_Epoch:
		requestedSlot = helpers.StartSlot(q.Epoch)
	case *ethpb.ListCommitteesRequest_Genesis:
		requestedSlot = 0
	default:
		requestedSlot = currentSlot
	}

	requestedEpoch := helpers.SlotToEpoch(requestedSlot)
	currentEpoch := helpers.SlotToEpoch(currentSlot)
	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information for an future epoch, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	committees := make(map[uint64]*ethpb.BeaconCommittees_CommitteesList)
	activeIndices := make([]uint64, 0)
	var err error
	if !featureconfig.Get().NewStateMgmt {
		committees, activeIndices, err = bs.retrieveCommitteesForEpochUsingOldArchival(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve committees for epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
	} else {
		committees, activeIndices, err = bs.retrieveCommitteesForEpoch(ctx, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				"Could not retrieve committees for epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
	}

	return &ethpb.BeaconCommittees{
		Epoch:                requestedEpoch,
		Committees:           committees,
		ActiveValidatorCount: uint64(len(activeIndices)),
	}, nil
}

func (bs *Server) retrieveCommitteesForEpoch(
	ctx context.Context,
	epoch uint64,
) (map[uint64]*ethpb.BeaconCommittees_CommitteesList, []uint64, error) {
	startSlot := helpers.StartSlot(epoch)
	requestedState, err := bs.StateGen.StateBySlot(ctx, startSlot)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get state")
	}
	seed, err := helpers.Seed(requestedState, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get seed")
	}
	activeIndices, err := helpers.ActiveValidatorIndices(requestedState, epoch)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get active indices")
	}

	committeesListsBySlot, err := computeCommittees(startSlot, activeIndices, seed)
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

// retrieveCommitteesForRoot uses the provided state root to get the current epoch committees.
// Note: This function is always recommended over retrieveCommitteesForEpoch as states are
// retrieved from the DB for this function, rather than generated.
func (bs *Server) retrieveCommitteesForRoot(
	ctx context.Context,
	root []byte,
) (map[uint64]*ethpb.BeaconCommittees_CommitteesList, []uint64, error) {
	requestedState, err := bs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get state")
	}
	epoch := helpers.CurrentEpoch(requestedState)
	seed, err := helpers.Seed(requestedState, epoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get seed")
	}
	activeIndices, err := helpers.ActiveValidatorIndices(requestedState, epoch)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, "Could not get active indices")
	}

	startSlot := helpers.StartSlot(epoch)
	committeesListsBySlot, err := computeCommittees(startSlot, activeIndices, seed)
	if err != nil {
		return nil, nil, status.Errorf(
			codes.InvalidArgument,
			"Could not compute committees for epoch %d: %v",
			epoch,
			err,
		)
	}
	return committeesListsBySlot, activeIndices, nil
}

func (bs *Server) retrieveCommitteesForEpochUsingOldArchival(
	ctx context.Context,
	epoch uint64,
) (map[uint64]*ethpb.BeaconCommittees_CommitteesList, []uint64, error) {
	var attesterSeed [32]byte
	var activeIndices []uint64
	var err error
	startSlot := helpers.StartSlot(epoch)
	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if helpers.SlotToEpoch(startSlot)+1 < currentEpoch {
		activeIndices, err = bs.HeadFetcher.HeadValidatorsIndices(ctx, helpers.SlotToEpoch(startSlot))
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
	} else if helpers.SlotToEpoch(startSlot)+1 == currentEpoch || helpers.SlotToEpoch(startSlot) == currentEpoch {
		// Otherwise, we use current beacon state to calculate the committees.
		requestedEpoch := helpers.SlotToEpoch(startSlot)
		activeIndices, err = bs.HeadFetcher.HeadValidatorsIndices(ctx, requestedEpoch)
		if err != nil {
			return nil, nil, status.Errorf(
				codes.Internal,
				"Could not retrieve active indices for requested epoch %d: %v",
				requestedEpoch,
				err,
			)
		}
		attesterSeed, err = bs.HeadFetcher.HeadSeed(ctx, requestedEpoch)
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
			currentEpoch,
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
