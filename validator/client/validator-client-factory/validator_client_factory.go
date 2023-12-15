package validator_client_factory

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	beaconApi "github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v4/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v4/validator/helpers"
)

func NewValidatorClient(ctx context.Context, validatorConn validatorHelpers.NodeConnection, opt ...beaconApi.ValidatorClientOpt) (iface.ValidatorClient, error) {
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		c := beaconApi.NewBeaconApiValidatorClient(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout(), opt...)
		if err := c.StartEventStream(ctx); err != nil {
			return nil, errors.Wrap(err, "could not start the validator client")
		}
		return c, nil
	} else {
		return grpcApi.NewGrpcValidatorClient(validatorConn.GetGrpcClientConn()), nil
	}
}
