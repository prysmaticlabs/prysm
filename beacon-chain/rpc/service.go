package rpc

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "rpc")

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	port     string
	listener net.Listener
}

// Config options for the beacon node RPC server.
type Config struct {
	Port string
}

// NewRPCService creates a new instance of a struct implementing the BeaconServiceServer
// interface.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		port:   cfg.Port,
	}
}

// Start the gRPC server.
func (s *Service) Start() {
	log.Info("Starting service")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		log.Errorf("Could not listen to port :%s: %v", s.port, err)
		return
	}
	s.listener = lis
	log.Infof("RPC server listening on port :%s", s.port)

	grpcServer := grpc.NewServer()
	pb.RegisterBeaconServiceServer(grpcServer, s)
	go func() {
		err = grpcServer.Serve(lis)
		if err != nil {
			log.Errorf("Could not serve gRPC: %v", err)
		}
	}()
}

// Stop the service.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// ShuffleValidators shuffles the validators into attesters/proposers.
func (s *Service) ShuffleValidators(ctx context.Context, req *pb.ShuffleRequest) (*pb.ShuffleResponse, error) {
	return nil, nil
}

// ProposeBlock is called by a proposer in a sharding client and a full beacon node
// the request into a beacon block that can then be included in a canonical chain.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	return nil, nil
}

// SignBlock is a function called by an attester in a sharding client to sign off
// on a block.
func (s *Service) SignBlock(ctx context.Context, req *pb.SignRequest) (*pb.SignResponse, error) {
	return nil, nil
}

// LatestBeaconHashHeight streams the latest beacon chain data.
func (s *Service) LatestBeaconHashHeight(req *empty.Empty, stream pb.BeaconService_LatestBeaconHashHeightServer) error {
	return stream.Send(nil)
}

// LatestCrystallizedState streams the latest beacon crystallized state.
func (s *Service) LatestCrystallizedState(req *empty.Empty, stream pb.BeaconService_LatestCrystallizedStateServer) error {
	return stream.Send(nil)
}
