// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"net"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "rpc")

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	port     string
	listener net.Listener
	withCert string
	withKey  string
}

// Config options for the beacon node RPC server.
type Config struct {
	Port     string
	CertFlag string
	KeyFlag  string
}

// NewRPCService creates a new instance of a struct implementing the BeaconServiceServer
// interface.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		port:     cfg.Port,
		withCert: cfg.CertFlag,
		withKey:  cfg.KeyFlag,
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

	var grpcServer *grpc.Server
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("could not load TLS keys: %s", err)
		}
		grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		log.Warn("You're on an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
		grpcServer = grpc.NewServer()
	}

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
//
// TODO: needs to properly shuffle.
func (s *Service) ShuffleValidators(req *pb.ShuffleRequest, stream pb.BeaconService_ShuffleValidatorsServer) error {
	return stream.Send(&pb.ShuffleResponse{IsProposer: true, IsAttester: false})
}

// ProposeBlock is called by a proposer in a sharding client and a full beacon node
// sends the request into a beacon block that can then be included in a canonical chain.
//
// TODO: needs implementation.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	return nil, nil
}

// SignBlock is a function called by an attester in a sharding client to sign off
// on a block.
//
// TODO: needs implementation.
func (s *Service) SignBlock(ctx context.Context, req *pb.SignRequest) (*pb.SignResponse, error) {
	return nil, nil
}
