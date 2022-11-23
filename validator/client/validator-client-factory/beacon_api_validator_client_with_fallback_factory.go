//go:build use_beacon_api_grpc_fallback
// +build use_beacon_api_grpc_fallback

package validator_client_factory

import (
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	grpcApi "github.com/prysmaticlabs/prysm/v3/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v3/validator/helpers"
)

func NewValidatorClient(validatorConn validatorHelpers.NodeConnection) iface.ValidatorClient {
	fallbackClient := grpcApi.NewGrpcValidatorClient(validatorConn.GetGrpcClientConn())
	return beaconApi.NewBeaconApiValidatorClientWithFallback(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout(), fallbackClient)
}
