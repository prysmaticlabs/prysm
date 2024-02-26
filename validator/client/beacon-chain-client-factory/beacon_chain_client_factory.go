package validator_client_factory

import (
	"github.com/prysmaticlabs/prysm/v5/config/features"
	beaconApi "github.com/prysmaticlabs/prysm/v5/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v5/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	nodeClientFactory "github.com/prysmaticlabs/prysm/v5/validator/client/node-client-factory"
	validatorHelpers "github.com/prysmaticlabs/prysm/v5/validator/helpers"
)

func NewBeaconChainClient(validatorConn validatorHelpers.NodeConnection, jsonRestHandler beaconApi.JsonRestHandler) iface.BeaconChainClient {
	grpcClient := grpcApi.NewGrpcBeaconChainClient(validatorConn.GetGrpcClientConn())
	if features.Get().EnableBeaconRESTApi {
		return beaconApi.NewBeaconApiBeaconChainClientWithFallback(jsonRestHandler, grpcClient)
	} else {
		return grpcClient
	}
}

func NewPrysmBeaconClient(validatorConn validatorHelpers.NodeConnection, jsonRestHandler beaconApi.JsonRestHandler) iface.PrysmBeaconChainClient {
	if features.Get().EnableBeaconRESTApi {
		return beaconApi.NewPrysmBeaconChainClient(jsonRestHandler, nodeClientFactory.NewNodeClient(validatorConn, jsonRestHandler))
	} else {
		return grpcApi.NewGrpcPrysmBeaconChainClient(validatorConn.GetGrpcClientConn())
	}
}
