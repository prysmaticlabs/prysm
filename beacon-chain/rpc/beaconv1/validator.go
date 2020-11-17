package beaconv1

import (
	"context"
	"errors"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

// GetValidator returns a validator specified by state and id or public key along with status and balance.
func (bs *Server) GetValidator(ctx context.Context, req *ethpb.StateValidatorRequest) (*ethpb.StateValidatorResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListValidators returns filterable list of validators with their balance, status and index.
func (bs *Server) ListValidators(ctx context.Context, req *ethpb.StateValidatorsRequest) (*ethpb.StateValidatorsResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListValidatorBalances returns a filterable list of validator balances.
func (bs *Server) ListValidatorBalances(ctx context.Context, req *ethpb.ValidatorBalancesRequest) (*ethpb.ValidatorBalancesResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListCommittees retrieves the committees for the given state at the given epoch.
func (bs *Server) ListCommittees(ctx context.Context, req *ethpb.StateCommitteesRequest) (*ethpb.StateCommitteesResponse, error) {
	return nil, errors.New("unimplemented")
}
