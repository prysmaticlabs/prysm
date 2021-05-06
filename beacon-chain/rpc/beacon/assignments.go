package beacon

import (
	"context"
	ptypes "github.com/gogo/protobuf/types"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"strconv"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errEpoch = "Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d"

// ListValidatorAssignments retrieves the validator assignments for a given epoch,
// optional validator indices or public keys may be included to filter validator assignments.
func (bs *Server) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	if int(req.PageSize) > cmd.Get().MaxRPCPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			cmd.Get().MaxRPCPageSize,
		)
	}

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	filtered := map[types.ValidatorIndex]bool{} // track filtered validators to prevent duplication in the response.
	filteredIndices := make([]types.ValidatorIndex, 0)
	var requestedEpoch types.Epoch
	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorAssignmentsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorAssignmentsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	currentEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	if requestedEpoch > currentEpoch {
		return nil, status.Errorf(
			codes.InvalidArgument,
			errEpoch,
			currentEpoch,
			requestedEpoch,
		)
	}

	startSlot, err := helpers.StartSlot(requestedEpoch)
	if err != nil {
		return nil, err
	}
	requestedState, err := bs.StateGen.StateBySlot(ctx, startSlot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve archived state for epoch %d: %v", requestedEpoch, err)
	}

	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := requestedState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if !ok {
			return nil, status.Errorf(codes.NotFound, "Could not find validator index for public key %#x", pubKey)
		}
		filtered[index] = true
		filteredIndices = append(filteredIndices, index)
	}

	// Filter out assignments by validator indices.
	for _, index := range req.Indices {
		if !filtered[index] {
			filteredIndices = append(filteredIndices, index)
		}
	}

	activeIndices, err := helpers.ActiveValidatorIndices(requestedState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve active validator indices: %v", err)
	}
	if len(filteredIndices) == 0 {
		if len(activeIndices) == 0 {
			return &ethpb.ValidatorAssignments{
				Assignments:   make([]*ethpb.ValidatorAssignments_CommitteeAssignment, 0),
				TotalSize:     int32(0),
				NextPageToken: strconv.Itoa(0),
			}, nil
		}
		// If no filter was specified, return assignments from active validator indices with pagination.
		filteredIndices = activeIndices
	}

	start, end, nextPageToken, err := pagination.StartAndEndPage(req.PageToken, int(req.PageSize), len(filteredIndices))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not paginate results: %v", err)
	}

	// Initialize all committee related data.
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(requestedState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	for _, index := range filteredIndices[start:end] {
		if uint64(index) >= uint64(requestedState.NumValidators()) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= validator count %d",
				index, requestedState.NumValidators())
		}
		comAssignment := committeeAssignments[index]
		pubkey := requestedState.PubkeyAtIndex(index)
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: comAssignment.Committee,
			CommitteeIndex:   comAssignment.CommitteeIndex,
			AttesterSlot:     comAssignment.AttesterSlot,
			ProposerSlots:    proposerIndexToSlots[index],
			PublicKey:        pubkey[:],
			ValidatorIndex:   index,
		}
		res = append(res, assign)
	}

	return &ethpb.ValidatorAssignments{
		Epoch:         requestedEpoch,
		Assignments:   res,
		NextPageToken: nextPageToken,
		TotalSize:     int32(len(filteredIndices)),
	}, nil
}

// NextEpochProposerList retrieves the validator assignments for a given epoch,
// [This api is specially used for Orchestrator client]
// optional validator indices or public keys may be included to filter validator assignments.
func (bs *Server) NextEpochProposerList(
	ctx context.Context, empty *ptypes.Empty) (*ethpb.ValidatorAssignments, error) {

	curEpoch := helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot())
	nextEpoch := curEpoch + 1

	curEpochStartSlot, err := helpers.StartSlot(curEpoch)
	if err != nil {
		return nil, err
	}

	nextEpochStratSlot, err := helpers.StartSlot(nextEpoch)
	if err != nil {
		return nil, err
	}

	// latestState is the state of last epoch.
	curEpochState, err := bs.StateGen.StateBySlot(ctx, curEpochStartSlot)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve archived state for epoch %d: %v", curEpoch, err)
	}

	// latestState is the state of last epoch.
	nextEpochState, err := bs.StateGen.StateBySlot(ctx, nextEpochStratSlot)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal, "Could not retrieve archived state for epoch %d: %v", nextEpoch, err)
	}

	res, err := prepareProposerAssignments(curEpochState, nextEpochState, curEpoch, nextEpoch)
	if err != nil {
		return nil, err
	}

	return &ethpb.ValidatorAssignments{
		Epoch:       curEpoch,
		Assignments: res,
	}, nil
}

// prepareProposerList
func prepareProposerAssignments(curEpochState, nextEpochState *stateTrie.BeaconState,
	curEpoch, nextEpoch types.Epoch) ([]*ethpb.ValidatorAssignments_CommitteeAssignment, error) {

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	// Initialize all committee related data.
	proposerListCurEpoch, err := helpers.ProposerAssignments(curEpochState, curEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments for current epoch: %v", err)
	}
	for index, proposerSlots := range proposerListCurEpoch {
		pubkey := curEpochState.PubkeyAtIndex(index)
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			ProposerSlots:  proposerSlots,
			PublicKey:      pubkey[:],
			ValidatorIndex: index,
		}
		res = append(res, assign)
	}

	proposerListNextEpoch, err := helpers.ProposerAssignments(nextEpochState, nextEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments for next epoch: %v", err)
	}
	for index, proposerSlots := range proposerListNextEpoch {
		pubkey := curEpochState.PubkeyAtIndex(index)
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			ProposerSlots:  proposerSlots,
			PublicKey:      pubkey[:],
			ValidatorIndex: index,
		}
		res = append(res, assign)
	}
	return res, nil
}
