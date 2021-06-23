package validator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetSyncMessageBlockRoot retrieves the sync committee block root of the beacon chain.
func (vs *Server) GetSyncMessageBlockRoot(ctx context.Context, _ *emptypb.Empty) (*prysmv2.SyncMessageBlockRootResponse, error) {
	r, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}

	return &prysmv2.SyncMessageBlockRootResponse{
		Root: r,
	}, nil
}

// SubmitSyncMessage submits the sync committee message to the network.
// It also saves the sync committee message into the pending pool for block inclusion.
func (vs *Server) SubmitSyncMessage(ctx context.Context, msg *prysmv2.SyncCommitteeMessage) (*emptypb.Empty, error) {
	errs, ctx := errgroup.WithContext(ctx)

	// Broadcasting and saving message into the pool in parallel. As one fail should not affect another.
	errs.Go(func() error {
		return vs.P2P.Broadcast(ctx, msg)
	})

	if err := vs.SyncCommitteePool.SaveSyncCommitteeMessage(msg); err != nil {
		return nil, err
	}

	// Wait for p2p broadcast to complete and return the first error (if any)
	err := errs.Wait()
	return nil, err
}

// GetSyncSubcommitteeIndex is called by a sync committee participant to get its subcommittee index for sync message aggregation duty.
func (vs *Server) GetSyncSubcommitteeIndex(ctx context.Context, req *prysmv2.SyncSubcommitteeIndexRequest) (*prysmv2.SyncSubcommitteeIndexRespond, error) {
	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if req.Slot > headState.Slot() {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots: %v", err)
		}
	}

	nextSlotEpoch := helpers.SlotToEpoch(headState.Slot() + 1)
	currentEpoch := helpers.CurrentEpoch(headState)

	switch {
	case altair.SyncCommitteePeriod(nextSlotEpoch) == altair.SyncCommitteePeriod(currentEpoch):
		committee, err := headState.CurrentSyncCommittee()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get current sync committee in head state: %v", err)
		}
		indices, err := helpers.CurrentEpochSyncSubcommitteeIndices(committee, bytesutil.ToBytes48(req.PublicKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get current sync subcommittee indices: %v", err)
		}
		return &prysmv2.SyncSubcommitteeIndexRespond{
			Indices: indices,
		}, nil
	// At sync committee period boundary, validator should sample the next epoch sync committee.
	case altair.SyncCommitteePeriod(nextSlotEpoch) == altair.SyncCommitteePeriod(currentEpoch)+1:
		committee, err := headState.NextSyncCommittee()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync committee in head state: %v", err)
		}
		indices, err := helpers.NextEpochSyncSubcommitteeIndices(committee, bytesutil.ToBytes48(req.PublicKey))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get next sync subcommittee indices: %v", err)
		}
		return &prysmv2.SyncSubcommitteeIndexRespond{
			Indices: indices,
		}, nil
	default:
		// Impossible condition.
		return nil, status.Errorf(codes.InvalidArgument, "Could get calculate sync subcommittee based on the period")
	}
}
