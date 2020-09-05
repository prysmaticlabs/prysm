package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconNodeConnection --
func (s *Server) GetBeaconNodeConnection(ctx context.Context, _ *ptypes.Empty) (*pb.NodeConnectionResponse, error) {
	syncStatus, err := s.syncChecker.Syncing(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "Could not determine sync status of beacon node")
	}
	return &pb.NodeConnectionResponse{
		BeaconNodeEndpoint: s.nodeGatewayEndpoint,
		Connected:          s.validatorService.Status() == nil,
		Syncing:            syncStatus,
	}, nil
}
