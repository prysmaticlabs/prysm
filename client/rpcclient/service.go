// Package rpcclient defines the services for the RPC connections of the client.
package rpcclient

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "rpc-client")

// Service for an RPCClient to a Beacon Node.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	conn     *grpc.ClientConn
	endpoint string
	withCert string
}

// Config for the RPCClient service.
type Config struct {
	Endpoint string
	CertFlag string
}

// NewRPCClient sets up a new beacon node RPC client connection.
func NewRPCClient(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: cfg.Endpoint,
		withCert: cfg.CertFlag,
	}
}

// Start the grpc connection.
func (s *Service) Start() {
	log.Info("Starting service")
	var server grpc.DialOption
	if s.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(s.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return
		}
		server = grpc.WithTransportCredentials(creds)
	} else {
		server = grpc.WithInsecure()
		log.Warn("You're on an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	conn, err := grpc.Dial(s.endpoint, server)
	if err != nil {
		log.Errorf("Could not connect to beacon node via RPC endpoint: %s: %v", s.endpoint, err)
		return
	}
	s.conn = conn
}

// Stop the dialed connection.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// BeaconServiceClient return the proto RPC interface.
func (s *Service) BeaconServiceClient() pb.BeaconServiceClient {
	return pb.NewBeaconServiceClient(s.conn)
}
