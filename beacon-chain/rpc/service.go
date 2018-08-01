package rpc

import (
	"context"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "rpc")

// Service hi.
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRPCService hi.
func NewRPCService(ctx context.Context) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start hi.
func (s *Service) Start() {
	log.Info("Starting RPC Service")
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Could not listen to port :8080: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBeaconServiceServer(grpcServer, s)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Could not serve: %v", err)
	}
}

// ShuffleValidators hi.
func (s *Service) ShuffleValidators(ctx context.Context, req *pb.ShuffleRequest) (*pb.ShuffleResponse, error) {
	return nil, nil
}

// ProposeBlock hi.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	return nil, nil
}

// SignBlock hi.
func (s *Service) SignBlock(ctx context.Context, req *pb.SignRequest) (*pb.SignResponse, error) {
	return nil, nil
}
