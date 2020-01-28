package beacon

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// GetBeaconConfig retrieves the current configuration parameters of the beacon chain.
func (bs *Server) GetBeaconConfig(ctx context.Context, req *ptypes.Empty) (*ethpb.BeaconConfig, error) {
	conf := make(map[string]*ptypes.Any)
	return &ethpb.BeaconConfig{
		Config: conf,
	}, nil
}
