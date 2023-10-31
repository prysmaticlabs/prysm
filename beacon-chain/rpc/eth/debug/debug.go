package debug

import (
	"context"

	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListForkChoiceHeadsV2 retrieves the leaves of the current fork choice tree.
func (ds *Server) ListForkChoiceHeadsV2(ctx context.Context, _ *emptypb.Empty) (*ethpbv2.ForkChoiceHeadsResponse, error) {
	ctx, span := trace.StartSpan(ctx, "debug.ListForkChoiceHeadsV2")
	defer span.End()

	headRoots, headSlots := ds.HeadFetcher.ChainHeads()
	resp := &ethpbv2.ForkChoiceHeadsResponse{
		Data: make([]*ethpbv2.ForkChoiceHead, len(headRoots)),
	}
	for i := range headRoots {
		isOptimistic, err := ds.OptimisticModeFetcher.IsOptimisticForRoot(ctx, headRoots[i])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not check if head is optimistic: %v", err)
		}
		resp.Data[i] = &ethpbv2.ForkChoiceHead{
			Root:                headRoots[i][:],
			Slot:                headSlots[i],
			ExecutionOptimistic: isOptimistic,
		}
	}

	return resp, nil
}

// GetForkChoice returns a dump fork choice store.
func (ds *Server) GetForkChoice(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.ForkChoiceDump, error) {
	return ds.ForkchoiceFetcher.ForkChoiceDump(ctx)
}
