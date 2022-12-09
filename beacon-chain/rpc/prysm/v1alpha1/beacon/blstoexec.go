package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SubmitBLSToExecutionChange receives a withdrawal credential change object via
// RPC and injects it into the beacon node's operations pool.
// Submission into this pool does not guarantee inclusion into a beacon block. If the object passes validation
// the node MUST broadcast it
func (bs *Server) SubmitBLSToExecutionChange(
	ctx context.Context,
	req *ethpb.SignedBLSToExecutionChange,
) (*ethpb.BLSToExecutionChangeResponse, error) {
	bs.BLSChangesPool.InsertBLSToExecChange(req)
	if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast SigledBLSToExecutionChange object: %v", err)
	}
	return nil, nil
}
