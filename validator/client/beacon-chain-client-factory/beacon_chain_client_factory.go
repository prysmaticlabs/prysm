package validator_client_factory

import (
	"github.com/prysmaticlabs/prysm/v4/config/features"
	beaconApi "github.com/prysmaticlabs/prysm/v4/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v4/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v4/validator/client/iface"
	nodeClientFactory "github.com/prysmaticlabs/prysm/v4/validator/client/node-client-factory"
	validatorHelpers "github.com/prysmaticlabs/prysm/v4/validator/helpers"
)

func NewBeaconChainClient(validatorConn validatorHelpers.NodeConnection) iface.BeaconChainClient {
	grpcClient := grpcApi.NewGrpcBeaconChainClient(validatorConn.GetGrpcClientConn())
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		return beaconApi.NewBeaconApiBeaconChainClientWithFallback(
			validatorConn.GetBeaconApiUrl(),
			validatorConn.GetBeaconApiTimeout(),
			grpcClient,
		)
	} else {
		return grpcClient
	}
}

func NewPrysmBeaconClient(validatorConn validatorHelpers.NodeConnection) iface.PrysmBeaconChainClient {
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		return beaconApi.NewPrysmBeaconChainClient(
			validatorConn.GetBeaconApiUrl(),
			validatorConn.GetBeaconApiTimeout(),
			nodeClientFactory.NewNodeClient(validatorConn),
		)
	} else {
		return grpcApi.NewGrpcPrysmBeaconChainClient(validatorConn.GetGrpcClientConn())
	}
}
