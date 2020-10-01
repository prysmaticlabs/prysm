package beacon_v1

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetStateRoot(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateRootResponse, error) {
	return nil, errors.New("unimplemented")
}

func GetStateFork(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateForkResponse, error) {
	return nil, errors.New("unimplemented")
}

func GetFinalityCheckpoints(ctx context.Context, req *ethpb.StateRequest) (*ethpb.StateFinalityCheckpointResponse, error) {
	return nil, errors.New("unimplemented")
}
