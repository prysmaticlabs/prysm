package rpc

import (
	"context"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
)

// GetBeaconNodeConnection retrieves the current beacon node connection
// information, as well as its sync status.
func (s *Server) GetBeaconNodeConnection(ctx context.Context, _ *ptypes.Empty) (*pb.NodeConnectionResponse, error) {
	genesis, err := s.genesisFetcher.GenesisInfo(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve genesis information: %v", err)
	}
	syncStatus, err := s.syncChecker.Syncing(ctx)
	if err != nil || s.validatorService.Status() != nil {
		return &pb.NodeConnectionResponse{
			GenesisTime:            uint64(time.Unix(genesis.GenesisTime.Seconds, 0).Unix()),
			DepositContractAddress: genesis.DepositContractAddress,
			BeaconNodeEndpoint:     s.nodeGatewayEndpoint,
			Connected:              false,
			Syncing:                false,
		}, nil
	}
	return &pb.NodeConnectionResponse{
		GenesisTime:            uint64(time.Unix(genesis.GenesisTime.Seconds, 0).Unix()),
		DepositContractAddress: genesis.DepositContractAddress,
		BeaconNodeEndpoint:     s.nodeGatewayEndpoint,
		Connected:              true,
		Syncing:                syncStatus,
	}, nil
}
