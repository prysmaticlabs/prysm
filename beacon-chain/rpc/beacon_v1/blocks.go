package beacon_v1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	return nil, errors.New("unimplemented")
}

func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	return nil, errors.New("unimplemented")
}

func (bs *Server) SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

func (bs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	return nil, errors.New("unimplemented")
}

func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockRootResponse, error) {
	return nil, errors.New("unimplemented")
}

func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	return nil, errors.New("unimplemented")
}
