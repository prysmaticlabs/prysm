package rpc

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) registerBeaconClient() {}

func (s *Server) GetBeaconStatus(ctx context.Context, _ *ptypes.Empty) (*pb.BeaconStatusResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) GetValidatorParticipation(
	ctx context.Context, req *ethpb.GetValidatorParticipationRequest,
) (*ethpb.ValidatorParticipationResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) GetValidatorPerformance(
	ctx context.Context, req *ethpb.ValidatorPerformanceRequest,
) (*ethpb.ValidatorPerformanceResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) GetValidatorBalances(
	ctx context.Context, req *ethpb.ListValidatorBalancesRequest,
) (*ethpb.ValidatorBalances, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) GetValidatorQueue(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.ValidatorQueue, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (s *Server) GetPeers(
	ctx context.Context, _ *ptypes.Empty,
) (*ethpb.Peers, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
