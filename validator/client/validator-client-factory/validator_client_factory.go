package validator_client_factory

import (
	"github.com/prysmaticlabs/prysm/v3/config/features"
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v3/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v3/validator/helpers"
)

func NewValidatorClient(validatorConn validatorHelpers.NodeConnection) iface.ValidatorClient {
	grpcClient := grpcApi.NewGrpcValidatorClient(validatorConn.GetGrpcClientConn())
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		return beaconApi.NewBeaconApiValidatorClientWithFallback(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout(), grpcClient)
	} else {
		return grpcClient
	}
}
