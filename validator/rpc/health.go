package rpc

import (
	"context"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetBeaconNodeConnection retrieves the current beacon node connection
// information, as well as its sync status.
func (s *Server) GetBeaconNodeConnection(ctx context.Context, _ *empty.Empty) (*pb.NodeConnectionResponse, error) {
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
func (s *Server) GetLogsEndpoints(ctx context.Context, _ *empty.Empty) (*pb.LogsEndpointResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// GetVersion --
func (s *Server) GetVersion(ctx context.Context, _ *empty.Empty) (*pb.VersionResponse, error) {
	beacon, err := s.beaconNodeClient.GetVersion(ctx, &ptypes.Empty{})
	if err != nil {
		return nil, err
	}

	return &pb.VersionResponse{
		Beacon:    beacon.Version,
		Validator: version.Version(),
	}, nil
}

// StreamBeaconLogs from the beacon node via a gRPC server-side stream.
func (s *Server) StreamBeaconLogs(req *empty.Empty, stream pb.Health_StreamBeaconLogsServer) error {
	// Wrap service context with a cancel in order to propagate the exiting of
	// this method properly to the beacon node server.
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	client, err := s.beaconNodeHealthClient.StreamBeaconLogs(ctx, req)
	if err != nil {
		return err
	}
	for {
		select {
		case <-s.ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		case <-client.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		default:
			resp, err := client.Recv()
			if err != nil {
				return errors.Wrap(err, "could not receive beacon logs from stream")
			}
			if err := stream.Send(resp); err != nil {
				return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
			}
		}
	}
}

// StreamValidatorLogs from the validator client via a gRPC server-side stream.
func (s *Server) StreamValidatorLogs(_ *empty.Empty, stream pb.Health_StreamValidatorLogsServer) error {
	ch := make(chan []byte, s.streamLogsBufferSize)
	sub := s.logsStreamer.LogsFeed().Subscribe(ch)
	defer func() {
		sub.Unsubscribe()
		defer close(ch)
	}()

	recentLogs := s.logsStreamer.GetLastFewLogs()
	logStrings := make([]string, len(recentLogs))
	for i, log := range recentLogs {
		logStrings[i] = string(log)
	}
	if err := stream.Send(&pb.LogsResponse{
		Logs: logStrings,
	}); err != nil {
		return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
	}
	for {
		select {
		case log := <-ch:
			resp := &pb.LogsResponse{
				Logs: []string{string(log)},
			}
			if err := stream.Send(resp); err != nil {
				return status.Errorf(codes.Unavailable, "Could not send over stream: %v", err)
			}
		case <-s.ctx.Done():
			return status.Error(codes.Canceled, "Context canceled")
		case err := <-sub.Err():
			return status.Errorf(codes.Canceled, "Subscriber error, closing: %v", err)
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Context canceled")
		}
	}
}
