package beacon

import (
	"context"
	"reflect"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// GetBeaconConfig returns the configuration of the beacon chain as understood by this node.
func (bs *Server) GetBeaconConfig(ctx context.Context, _ *ptypes.Empty) (*ethpb.BeaconConfig, error) {
	conf := params.BeaconConfig()
	val := reflect.ValueOf(conf).Elem()
	numFields := val.Type().NumField()
	res := make(map[string]*ptypes.Any, numFields)
	for i := 0; i < numFields; i++ {
		res[val.Type().Field(i).Name] = &ptypes.Any{
			TypeUrl:
		}
	}
	return &ethpb.BeaconConfig{
		Config:               res,
	}, nil
}
