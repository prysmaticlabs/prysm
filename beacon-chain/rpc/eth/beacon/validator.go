package beacon

import (
	"context"

	corehelpers "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListCommittees retrieves the committees for the given state at the given epoch.
// If the requested slot and index are defined, only those committees are returned.
func (bs *Server) ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListCommittees")
	defer span.End()

	st, err := bs.Stater.State(ctx, req.StateId)
	if err != nil {
		return nil, helpers.PrepareStateFetchGRPCError(err)
	}

	epoch := slots.ToEpoch(st.Slot())
	if req.Epoch != nil {
		epoch = *req.Epoch
	}
	activeCount, err := corehelpers.ActiveValidatorCount(ctx, st, epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get active validator count: %v", err)
	}

	startSlot, err := slots.EpochStart(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	endSlot, err := slots.EpochEnd(epoch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid epoch: %v", err)
	}
	committeesPerSlot := corehelpers.SlotCommitteeCount(activeCount)
	committees := make([]*ethpb.Committee, 0)
	for slot := startSlot; slot <= endSlot; slot++ {
		if req.Slot != nil && slot != *req.Slot {
			continue
		}
		for index := primitives.CommitteeIndex(0); index < primitives.CommitteeIndex(committeesPerSlot); index++ {
			if req.Index != nil && index != *req.Index {
				continue
			}
			committee, err := corehelpers.BeaconCommitteeFromState(ctx, st, slot, index)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not get committee: %v", err)
			}
			committeeContainer := &ethpb.Committee{
				Index:      index,
				Slot:       slot,
				Validators: committee,
			}
			committees = append(committees, committeeContainer)
		}
	}

	isOptimistic, err := helpers.IsOptimistic(ctx, req.StateId, bs.OptimisticModeFetcher, bs.Stater, bs.ChainInfoFetcher, bs.BeaconDB)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not check if slot's block is optimistic: %v", err)
	}

	blockRoot, err := st.LatestBlockHeader().HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate root of latest block header")
	}
	isFinalized := bs.FinalizationFetcher.IsFinalized(ctx, blockRoot)

	return &ethpb.StateCommitteesResponse{Data: committees, ExecutionOptimistic: isOptimistic, Finalized: isFinalized}, nil
}
