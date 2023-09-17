package iface

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/validator"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

var ErrNotSupported = errors.New("endpoint not supported")

type ValidatorCount struct {
	Status string
	Count  uint64
}

type ValidatorCountProvider interface {
	GetValidatorCount(context.Context, string, []validator.ValidatorStatus) ([]ValidatorCount, error)
}

type BeaconChainClient interface {
	GetChainHead(ctx context.Context, in *empty.Empty) (*ethpb.ChainHead, error)
	ListValidatorBalances(ctx context.Context, in *ethpb.ListValidatorBalancesRequest) (*ethpb.ValidatorBalances, error)
	ListValidators(ctx context.Context, in *ethpb.ListValidatorsRequest) (*ethpb.Validators, error)
	GetValidatorQueue(ctx context.Context, in *empty.Empty) (*ethpb.ValidatorQueue, error)
	GetValidatorPerformance(ctx context.Context, in *ethpb.ValidatorPerformanceRequest) (*ethpb.ValidatorPerformanceResponse, error)
	GetValidatorParticipation(ctx context.Context, in *ethpb.GetValidatorParticipationRequest) (*ethpb.ValidatorParticipationResponse, error)
}

// MockBeaconChainClientComposed is the interface used by mockgen to generate a type which
// implements BeaconChainClient and ValidatorCountProvider interfaces.
type MockBeaconChainClientComposed interface {
	BeaconChainClient
	ValidatorCountProvider
}
