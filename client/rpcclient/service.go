package rpcclient

import (
	"context"
	"fmt"
	"net/url"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "rpc-client")

// Service for an RPCClient to a Beacon Node.
type Service struct {
	ctx      context.Context
	cancel   context.CancelFunc
	conn     *grpc.ClientConn
	endpoint string
}

// Config for the RPCClient service.
type Config struct {
	Endpoint string
}

// NewRPCClient sets up a new beacon node RPC client connection.
func NewRPCClient(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:      ctx,
		cancel:   cancel,
		endpoint: cfg.Endpoint,
	}
}

// Start the grpc connection.
func (s *Service) Start() {
	log.Info("Starting service")
	endpointURL, err := url.Parse(s.endpoint)
	if err != nil {
		log.Fatalf("Could not parse endpoint URL: %s, %v", s.endpoint, err)
		return
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", endpointURL.Hostname(), endpointURL.Port()), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect to beacon node via RPC endpoint: %s: %v", s.endpoint, err)
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
