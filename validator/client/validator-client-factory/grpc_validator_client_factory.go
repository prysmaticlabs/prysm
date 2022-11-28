//go:build !use_beacon_api
// +build !use_beacon_api

package validator_client_factory

import (
	grpcApi "github.com/prysmaticlabs/prysm/v3/validator/client/grpc-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v3/validator/helpers"
)

func NewValidatorClient(validatorConn validatorHelpers.NodeConnection) iface.ValidatorClient {
	return grpcApi.NewGrpcValidatorClient(validatorConn.GetGrpcClientConn())
}
