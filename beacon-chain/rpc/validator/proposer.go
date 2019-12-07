package validator

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (vs *Server) GetBlock(context.Context, *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

// ProposeBlock is a skeleton for the ProposeBlock GRPC call.
func (vs *Server) ProposeBlock(context.Context, *ethpb.BlockRequest) (*ptypes.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}
