package beacon

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetStateRoot(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateRootResponse, error) {
	return &ethpb.StateRootResponse{
		StateRoot: []byte{},
	}, nil
}

func GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	return &ethpb.StateForkResponse{}, nil
}

func GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	return &ethpb.StateFinalityCheckpointResponse{}, nil
}
