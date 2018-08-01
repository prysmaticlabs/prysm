package rpc

import (
	"context"
	"fmt"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "rpc")

// Service hi.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	port     string
	listener net.Listener
}

// Config options for the beacon RPC server.
type Config struct {
	Port string
}

// NewRPCService hi.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
		port:   cfg.Port,
	}
}

// Start hi.
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

// ShuffleValidators hi.
func (s *Service) ShuffleValidators(req *pb.ShuffleRequest, stream pb.BeaconService_ShuffleValidatorsServer) error {
	return stream.Send(&pb.ShuffleResponse{IsProposer: true, IsAttester: false})
}

// ProposeBlock hi.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	return nil, nil
}

// SignBlock hi.
func (s *Service) SignBlock(ctx context.Context, req *pb.SignRequest) (*pb.SignResponse, error) {
	return nil, nil
}
