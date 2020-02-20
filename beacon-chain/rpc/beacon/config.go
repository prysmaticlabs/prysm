package beacon

import (
	"context"
	"fmt"
	"reflect"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GetBeaconConfig retrieves the current configuration parameters of the beacon chain.
func (bs *Server) GetBeaconConfig(ctx context.Context, _ *ptypes.Empty) (*ethpb.BeaconConfig, error) {
	conf := params.BeaconConfig()
	val := reflect.ValueOf(conf).Elem()
	numFields := val.Type().NumField()
	res := make(map[string]string, numFields)
	for i := 0; i < numFields; i++ {
		res[val.Type().Field(i).Name] = fmt.Sprintf("%v", val.Field(i).Interface())
	}
	return &ethpb.BeaconConfig{
		Config: res,
	}, nil
}
