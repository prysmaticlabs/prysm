package rpc

import (
	"context"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"google.golang.org/grpc"
)

type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func NewRPCService(ctx context.Context) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Service) Start() {
	log.Info("Starting RPC Service")
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Could not listen to port :8080: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterBeaconChainServer(grpcServer, s)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("Could not serve: %v", err)
	}
}
