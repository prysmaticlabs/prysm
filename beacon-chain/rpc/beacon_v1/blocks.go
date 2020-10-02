package beacon_v1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

// GetBlockHeader retrieves block header for given block id.
func (bs *Server) GetBlockHeader(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockHeaderResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListBlockHeaders retrieves block headers matching given query. By default it will fetch current head slot blocks.
func (bs *Server) ListBlockHeaders(ctx context.Context, req *ethpb.BlockHeadersRequest) (*ethpb.BlockHeadersResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitBlock instructs the beacon node to broadcast a newly signed beacon block to the beacon network, to be
// included in the beacon chain. The beacon node is not required to validate the signed BeaconBlock, and a successful
// response (20X) only indicates that the broadcast has been successful. The beacon node is expected to integrate the
// new block into its state, and therefore validate the block internally, however blocks which fail the validation are
// still broadcast but a different status code is returned (202).
func (bs *Server) SubmitBlock(ctx context.Context, req *ethpb.BeaconBlockContainer) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// GetBlock retrieves block details for given block id.
func (bs *Server) GetBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetBlockRoot retrieves hashTreeRoot of BeaconBlock/BeaconBlockHeader.
func (bs *Server) GetBlockRoot(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockRootResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListBlockAttestations retrieves attestation included in requested block.
func (bs *Server) ListBlockAttestations(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BlockAttestationsResponse, error) {
	return nil, errors.New("unimplemented")
}
