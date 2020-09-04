package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
)

// GetBeaconNodeConnection --
func (s *Server) GetBeaconNodeConnection(ctx context.Context, _ *ptypes.Empty) (*pb.NodeConnectionResponse, error) {
	syncStatus, err := s.syncChecker.Syncing(ctx)
	if err != nil || s.validatorService.Status() != nil {
		return &pb.NodeConnectionResponse{
			BeaconNodeEndpoint: s.nodeGatewayEndpoint,
			Connected:          false,
			Syncing:            false,
		}, nil
	}
	return &pb.NodeConnectionResponse{
		BeaconNodeEndpoint: s.nodeGatewayEndpoint,
		Connected:          true,
		Syncing:            syncStatus,
	}, nil
}
