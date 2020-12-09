package rpc

import (
	"context"
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
)

// GetBeaconNodeConnection retrieves the current beacon node connection
// information, as well as its sync status.
func (s *Server) GetBeaconNodeConnection(ctx context.Context, _ *ptypes.Empty) (*pb.NodeConnectionResponse, error) {
	syncStatus, err := s.syncChecker.Syncing(ctx)
	if err != nil || s.validatorService.Status() != nil {
		return &pb.NodeConnectionResponse{
			GenesisTime:        0,
			BeaconNodeEndpoint: s.nodeGatewayEndpoint,
			Connected:          false,
			Syncing:            false,
		}, nil
	}
	genesis, err := s.genesisFetcher.GenesisInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.NodeConnectionResponse{
		GenesisTime:            uint64(time.Unix(genesis.GenesisTime.Seconds, 0).Unix()),
		DepositContractAddress: genesis.DepositContractAddress,
		BeaconNodeEndpoint:     s.nodeGatewayEndpoint,
		Connected:              true,
		Syncing:                syncStatus,
	}, nil
}

// GetLogsEndpoints for the beacon and validator client.
func (s *Server) GetLogsEndpoints(ctx context.Context, _ *ptypes.Empty) (*pb.LogsEndpointResponse, error) {
	beaconLogsEndpoint, err := s.beaconNodeInfoFetcher.BeaconLogsEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.LogsEndpointResponse{
		BeaconLogsEndpoint:    beaconLogsEndpoint + "/logs",
		ValidatorLogsEndpoint: fmt.Sprintf("%s:%d/logs", s.validatorMonitoringHost, s.validatorMonitoringPort),
	}, nil
}
