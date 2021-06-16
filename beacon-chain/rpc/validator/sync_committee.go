package validator

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetSyncSubcommitteeIndex is called by a sync committee participant to get its subcommittee index for aggregation duty.
func (vs *Server) GetSyncSubcommitteeIndex(ctx context.Context, req *ethpb.SyncSubcommitteeIndexRequest) (*ethpb.SyncSubcommitteeIndexRespond, error) {
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

	committee, err := headState.CurrentSyncCommittee()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync committee in head state: %v", err)
	}
	indices, err := helpers.CurrentEpochSyncSubcommitteeIndices(committee, bytesutil.ToBytes48(req.PublicKey))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get sync subcommittee indices: %v", err)
	}

	return &ethpb.SyncSubcommitteeIndexRespond{
		Indices: indices,
	}, nil
}
