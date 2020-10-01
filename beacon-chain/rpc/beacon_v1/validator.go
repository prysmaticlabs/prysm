package beacon_v1

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

func GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	return nil, errors.New("unimplemented")
}

func ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	return nil, errors.New("unimplemented")
}

func ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	return nil, errors.New("unimplemented")
}
