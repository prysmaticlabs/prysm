//go:build use_beacon_api
// +build use_beacon_api

package validator_client_factory

import (
	beaconApi "github.com/prysmaticlabs/prysm/v3/validator/client/beacon-api"
	"github.com/prysmaticlabs/prysm/v3/validator/client/iface"
	"google.golang.org/grpc"
)

func NewValidatorClient(cc grpc.ClientConnInterface) iface.ValidatorClient {
	return beaconApi.NewBeaconApiValidatorClient()
}
