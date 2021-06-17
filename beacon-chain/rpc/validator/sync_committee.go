package validator

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetSyncMessageBlockRoot retrieves the sync committee block root of the beacon chain.
func (vs *Server) GetSyncMessageBlockRoot(ctx context.Context, req *ethpb.SyncMessageBlockRootRequest) (*ethpb.SyncMessageBlockRootResponse, error) {
	// Prevent underflow from requested slot.
	slot := types.Slot(0)
	if req.Slot > 1 {
		slot = req.Slot - 1
	}
	// Short cut, where copying state and processing slots are not required.
	if slot == vs.HeadFetcher.HeadSlot() {
		r, err := vs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
		}
		return &ethpb.SyncMessageBlockRootResponse{
			Root: r,
		}, nil
	}

	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if req.Slot > headState.Slot() {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not not process slots: %v", err)
		}
	}
	r, err := helpers.BlockRootAtSlot(headState, slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not calculate block root: %v", err)
	}

	return &ethpb.SyncMessageBlockRootResponse{
		Root: r,
	}, nil
}

// SubmitSyncMessage submits the sync committee message to the network.
// It also saves the sync committee message into the pending pool for block inclusion.
func (vs *Server) SubmitSyncMessage(ctx context.Context, msg *ethpb.SyncCommitteeMessage) (*emptypb.Empty, error) {
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
