package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
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

// ShuffleValidators shuffles the validators into attesters/proposers. This function is not a stream but
// rather an on-demand RPC method as an attester/proposer determines its time to
// perform its responsibility within its respective event loop.
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

// LatestBeaconBlock streams the latest beacon chain data.
func (s *Service) LatestBeaconBlock(req *empty.Empty, stream pb.BeaconService_LatestBeaconBlockServer) error {
	delayChan := time.NewTicker(time.Second * 5).C
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-delayChan:
			block, err := types.NewGenesisBlock()
			if err != nil {
				return fmt.Errorf("could not create block: %v", block)
			}
			if err := stream.Send(block.Proto()); err != nil {
				return fmt.Errorf("could not send latest beacon block via stream: %v", err)
			}
		}
	}
}

// LatestCrystallizedState streams the latest beacon crystallized state.
func (s *Service) LatestCrystallizedState(req *empty.Empty, stream pb.BeaconService_LatestCrystallizedStateServer) error {
	delayChan := time.NewTicker(time.Second * 5).C
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-delayChan:
			_, state := types.NewGenesisStates()
			if err := stream.Send(state.Proto()); err != nil {
				return fmt.Errorf("could not send latest crystallized state via stream: %v", err)
			}
		}
	}
}
