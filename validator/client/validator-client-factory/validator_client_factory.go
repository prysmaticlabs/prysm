package validator_client_factory

import (
	"github.com/prysmaticlabs/prysm/v3/config/features"
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v3/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v3/validator/helpers"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "powchain")

func NewValidatorClient(validatorConn validatorHelpers.NodeConnection) iface.ValidatorClient {
	grpcClient := grpcApi.NewGrpcValidatorClient(validatorConn.GetGrpcClientConn())
	featureFlags := features.Get()

	if featureFlags.EnableBeaconRESTApi {
		log.Error("******************BEACON REST API ENABLED!!!")
		return beaconApi.NewBeaconApiValidatorClientWithFallback(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout(), grpcClient)
	} else {
		log.Error("******************BEACON REST API NOT ENABLED!!!")
		return grpcClient
	}
}
