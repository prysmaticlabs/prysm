package beacon

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	return &ethpb.BlockHeaderResponse{}, nil
}

func ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	return &ethpb.BlockHeadersResponse{}, nil
}

func SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	return &ptypes.Empty{}, nil
}

func GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	return &ethpb.BlockResponse{}, nil
}

func GetBlockRoot(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockRootResponse, error) {
	return &ethpb.BlockRootResponse{}, nil
}

func ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	return &ethpb.BlockAttestationsResponse{}, nil
}
