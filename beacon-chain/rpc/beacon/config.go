package beacon

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconConfig returns the configuration of the beacon chain as understood by this node.
func (bs *Server) GetBeaconConfig(ctx context.Context, _ *ptypes.Empty) (*ethpb.BeaconConfig, error) {
	return nil, status.Error(codes.Internal, "not implemented")
}
