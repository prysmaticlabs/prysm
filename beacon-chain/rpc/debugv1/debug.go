package debugv1

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetState returns the full beacon state for a given state id.
func (bs *Server) GetState(ctx context.Context, req *ethpb.StateRequest) (*ethpb.BeaconStateResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetForkChoiceHead retrieves the fork choice leaves for the current head.
func (bs *Server) GetForkChoiceHeads(ctx context.Context, _ *emptypb.Empty) (*ethpb.ForkChoiceHeadsResponse, error) {
	return nil, errors.New("unimplemented")
}
