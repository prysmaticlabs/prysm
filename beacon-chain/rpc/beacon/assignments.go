package beacon

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/pagination"
	"github.com/prysmaticlabs/prysm/shared/params"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListValidatorAssignments retrieves the validator assignments for a given epoch,
// optional validator indices or public keys may be included to filter validator assignments.
func (bs *Server) ListValidatorAssignments(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	if int(req.PageSize) > flags.Get().MaxPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			flags.Get().MaxPageSize,
		)
	}

	if !featureconfig.Get().NewStateMgmt {
		return bs.listValidatorAssignmentsUsingOldArchival(ctx, req)
	}

	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.
	filteredIndices := make([]uint64, 0)
	var requestedEpoch uint64
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
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			currentEpoch,
			requestedEpoch,
		)
	}

	requestedState, err := bs.StateGen.StateBySlot(ctx, helpers.StartSlot(requestedEpoch))
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
	committeeAssignments := map[uint64]*helpers.CommitteeAssignmentContainer{}
	proposerIndexToSlots := make(map[uint64][]uint64)
	committeeAssignments, proposerIndexToSlots, err = helpers.CommitteeAssignments(requestedState, requestedEpoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	for _, index := range filteredIndices[start:end] {
		if index >= uint64(requestedState.NumValidators()) {
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

func (bs *Server) listValidatorAssignmentsUsingOldArchival(
	ctx context.Context, req *ethpb.ListValidatorAssignmentsRequest,
) (*ethpb.ValidatorAssignments, error) {
	var res []*ethpb.ValidatorAssignments_CommitteeAssignment
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not get head state")
	}
	filtered := map[uint64]bool{} // track filtered validators to prevent duplication in the response.
	filteredIndices := make([]uint64, 0)
	requestedEpoch := helpers.CurrentEpoch(headState)

	switch q := req.QueryFilter.(type) {
	case *ethpb.ListValidatorAssignmentsRequest_Genesis:
		if q.Genesis {
			requestedEpoch = 0
		}
	case *ethpb.ListValidatorAssignmentsRequest_Epoch:
		requestedEpoch = q.Epoch
	}

	if requestedEpoch > helpers.CurrentEpoch(headState) {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot retrieve information about an epoch in the future, current epoch %d, requesting %d",
			helpers.CurrentEpoch(headState),
			requestedEpoch,
		)
	}

	// Filter out assignments by public keys.
	for _, pubKey := range req.PublicKeys {
		index, ok := headState.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
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

	activeIndices, err := helpers.ActiveValidatorIndices(headState, requestedEpoch)
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

	shouldFetchFromArchive := requestedEpoch < bs.FinalizationFetcher.FinalizedCheckpt().Epoch

	// initialize all committee related data.
	committeeAssignments := map[uint64]*helpers.CommitteeAssignmentContainer{}
	proposerIndexToSlots := make(map[uint64][]uint64)
	archivedInfo := &pb.ArchivedCommitteeInfo{}
	archivedBalances := make([]uint64, 0)
	archivedAssignments := make(map[uint64]*ethpb.ValidatorAssignments_CommitteeAssignment)

	if shouldFetchFromArchive {
		archivedInfo, archivedBalances, err = bs.archivedCommitteeData(ctx, requestedEpoch)
		if err != nil {
			return nil, err
		}
		archivedAssignments, err = archivedValidatorCommittee(
			requestedEpoch,
			archivedInfo,
			activeIndices,
			archivedBalances,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve archived assignment for epoch %d: %v", requestedEpoch, err)
		}
	} else {
		committeeAssignments, proposerIndexToSlots, err = helpers.CommitteeAssignments(headState, requestedEpoch)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
		}
	}

	for _, index := range filteredIndices[start:end] {
		if index >= uint64(headState.NumValidators()) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= validator count %d",
				index, headState.NumValidators())
		}
		if shouldFetchFromArchive {
			assignment, ok := archivedAssignments[index]
			if !ok {
				return nil, status.Errorf(codes.Internal, "Could not get archived committee assignment for index %d", index)
			}
			pubkey := headState.PubkeyAtIndex(index)
			assignment.PublicKey = pubkey[:]
			res = append(res, assignment)
			continue
		}
		comAssignment := committeeAssignments[index]
		pubkey := headState.PubkeyAtIndex(index)
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

// Computes validator assignments for an epoch and validator index using archived committee
// information, archived balances, and a set of active validators.
func archivedValidatorCommittee(
	epoch uint64,
	archivedInfo *pb.ArchivedCommitteeInfo,
	activeIndices []uint64,
	archivedBalances []uint64,
) (map[uint64]*ethpb.ValidatorAssignments_CommitteeAssignment, error) {
	proposerSeed := bytesutil.ToBytes32(archivedInfo.ProposerSeed)
	attesterSeed := bytesutil.ToBytes32(archivedInfo.AttesterSeed)

	startSlot := helpers.StartSlot(epoch)
	proposerIndexToSlots := make(map[uint64][]uint64)
	activeVals := make([]*ethpb.Validator, len(archivedBalances))
	for i, bal := range archivedBalances {
		activeVals[i] = &ethpb.Validator{EffectiveBalance: bal}
	}

	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		seedWithSlot := append(proposerSeed[:], bytesutil.Bytes8(slot)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		i, err := helpers.ComputeProposerIndexWithValidators(activeVals, activeIndices, seedWithSlotHash)
		if err != nil {
			return nil, errors.Wrapf(err, "could not check proposer at slot %d", slot)
		}
		proposerIndexToSlots[i] = append(proposerIndexToSlots[i], slot)
	}

	assignmentMap := make(map[uint64]*ethpb.ValidatorAssignments_CommitteeAssignment)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		for i := uint64(0); i < countAtSlot; i++ {
			committee, err := helpers.BeaconCommittee(activeIndices, attesterSeed, slot, i)
			if err != nil {
				return nil, errors.Wrap(err, "could not compute committee")
			}
			for _, index := range committee {
				assignmentMap[index] = &ethpb.ValidatorAssignments_CommitteeAssignment{
					BeaconCommittees: committee,
					CommitteeIndex:   i,
					AttesterSlot:     slot,
					ProposerSlots:    proposerIndexToSlots[index],
				}
			}
		}
	}
	return assignmentMap, nil
}

func (bs *Server) archivedCommitteeData(ctx context.Context, requestedEpoch uint64) (*pb.ArchivedCommitteeInfo,
	[]uint64, error) {
	archivedInfo, err := bs.BeaconDB.ArchivedCommitteeInfo(ctx, requestedEpoch)
	if err != nil {
		return nil, nil, status.Errorf(
			codes.Internal,
			"Could not retrieve archived committee info for epoch %d",
			requestedEpoch,
		)
	}
	if archivedInfo == nil {
		return nil, nil, status.Errorf(
			codes.NotFound,
			"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
			requestedEpoch,
		)
	}
	archivedBalances, err := bs.BeaconDB.ArchivedBalances(ctx, requestedEpoch)
	if err != nil {
		return nil, nil, status.Errorf(
			codes.Internal,
			"Could not retrieve archived balances for epoch %d",
			requestedEpoch,
		)
	}
	if archivedBalances == nil {
		return nil, nil, status.Errorf(
			codes.NotFound,
			"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
			requestedEpoch,
		)
	}
	return archivedInfo, archivedBalances, nil
}
