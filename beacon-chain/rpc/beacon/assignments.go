package beacon

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
	if int(req.PageSize) > params.BeaconConfig().MaxPageSize {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Requested page size %d can not be greater than max size %d",
			req.PageSize,
			params.BeaconConfig().MaxPageSize,
		)
	}

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
		index, ok, err := bs.BeaconDB.ValidatorIndex(ctx, bytesutil.ToBytes48(pubKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve validator index: %v", err)
		}
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

	for _, index := range filteredIndices[start:end] {
		if int(index) >= len(headState.Validators) {
			return nil, status.Errorf(codes.OutOfRange, "Validator index %d >= validator count %d",
				index, len(headState.Validators))
		}
		if shouldFetchFromArchive {
			archivedInfo, err := bs.BeaconDB.ArchivedCommitteeInfo(ctx, requestedEpoch)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"Could not retrieve archived committee info for epoch %d",
					requestedEpoch,
				)
			}
			if archivedInfo == nil {
				return nil, status.Errorf(
					codes.NotFound,
					"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
					requestedEpoch,
				)
			}
			archivedBalances, err := bs.BeaconDB.ArchivedBalances(ctx, requestedEpoch)
			if err != nil {
				return nil, status.Errorf(
					codes.Internal,
					"Could not retrieve archived balances for epoch %d",
					requestedEpoch,
				)
			}
			if archivedBalances == nil {
				return nil, status.Errorf(
					codes.NotFound,
					"Could not retrieve data for epoch %d, perhaps --archive in the running beacon node is disabled",
					requestedEpoch,
				)
			}
			committee, committeeIndex, attesterSlot, proposerSlot, err := archivedValidatorCommittee(
				requestedEpoch,
				index,
				archivedInfo,
				activeIndices,
				archivedBalances,
			)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve archived assignment for validator %d: %v", index, err)
			}
			assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
				BeaconCommittees: committee,
				CommitteeIndex:   committeeIndex,
				AttesterSlot:     attesterSlot,
				ProposerSlot:     proposerSlot,
				PublicKey:        headState.Validators[index].PublicKey,
			}
			res = append(res, assign)
			continue
		}
		committee, committeeIndex, attesterSlot, proposerSlot, err := helpers.CommitteeAssignment(headState, requestedEpoch, index)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve assignment for validator %d: %v", index, err)
		}
		assign := &ethpb.ValidatorAssignments_CommitteeAssignment{
			BeaconCommittees: committee,
			CommitteeIndex:   committeeIndex,
			AttesterSlot:     attesterSlot,
			ProposerSlot:     proposerSlot,
			PublicKey:        headState.Validators[index].PublicKey,
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
	validatorIndex uint64,
	archivedInfo *pb.ArchivedCommitteeInfo,
	activeIndices []uint64,
	archivedBalances []uint64,
) ([]uint64, uint64, uint64, uint64, error) {
	proposerSeed := bytesutil.ToBytes32(archivedInfo.ProposerSeed)
	attesterSeed := bytesutil.ToBytes32(archivedInfo.AttesterSeed)

	startSlot := helpers.StartSlot(epoch)
	proposerIndexToSlot := make(map[uint64]uint64)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		seedWithSlot := append(proposerSeed[:], bytesutil.Bytes8(slot)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		i, err := archivedProposerIndex(activeIndices, archivedBalances, seedWithSlotHash)
		if err != nil {
			return nil, 0, 0, 0, errors.Wrapf(err, "could not check proposer at slot %d", slot)
		}
		proposerIndexToSlot[i] = slot
	}
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
				return nil, 0, 0, 0, errors.Wrap(err, "could not compute committee")
			}
			for _, index := range committee {
				if validatorIndex == index {
					proposerSlot, _ := proposerIndexToSlot[validatorIndex]
					return committee, i, slot, proposerSlot, nil
				}
			}
		}
	}
	return nil, 0, 0, 0, fmt.Errorf("could not find committee for validator index %d", validatorIndex)
}

func archivedProposerIndex(activeIndices []uint64, activeBalances []uint64, seed [32]byte) (uint64, error) {
	length := uint64(len(activeIndices))
	if length == 0 {
		return 0, errors.New("empty indices list")
	}
	maxRandomByte := uint64(1<<8 - 1)
	for i := uint64(0); ; i++ {
		candidateIndex, err := helpers.ComputeShuffledIndex(i%length, length, seed, true)
		if err != nil {
			return 0, err
		}
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashutil.Hash(b)[i%32]
		effectiveBalance := activeBalances[candidateIndex]
		if effectiveBalance >= params.BeaconConfig().MaxEffectiveBalance {
			// if the actual balance is greater than or equal to the max effective balance,
			// we just determine the proposer index using config.MaxEffectiveBalance.
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		if effectiveBalance*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}
