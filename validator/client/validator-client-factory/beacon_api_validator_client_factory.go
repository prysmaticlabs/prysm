//go:build use_beacon_api && !use_beacon_api_grpc_fallback

package validator_client_factory

import (
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	validatorHelpers "github.com/prysmaticlabs/prysm/v3/validator/helpers"
)

func NewValidatorClient(validatorConn validatorHelpers.NodeConnection) iface.ValidatorClient {
	return beaconApi.NewBeaconApiValidatorClient(validatorConn.GetBeaconApiUrl(), validatorConn.GetBeaconApiTimeout())
}
