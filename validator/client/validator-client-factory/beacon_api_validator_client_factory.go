//go:build use_beacon_api
// +build use_beacon_api

package validator_client_factory

import (
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
)

func NewValidatorClient(validatorConn *ValidatorConnection) iface.ValidatorClient {
	return beaconApi.NewBeaconApiValidatorClient(validatorConn.BeaconApiConn.Url, validatorConn.BeaconApiConn.Timeout)
}
