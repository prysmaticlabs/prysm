package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	currentSlot := bs.GenesisTimeFetcher.CurrentSlot()
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListCommitteesRequest_Epoch:
		startSlot = helpers.StartSlot(q.Epoch)
	case *ethpb.ListCommitteesRequest_Genesis:
		requestingGenesis = q.Genesis
		if !requestingGenesis {
			startSlot = currentSlot
		}
	default:
		startSlot = currentSlot
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
