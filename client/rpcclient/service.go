package rpcclient

import (
	"context"

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
	conn, err := grpc.Dial(s.endpoint, grpc.WithInsecure())
	if err != nil {
		log.Errorf("Could not connect to beacon node via RPC endpoint: %s: %v", s.endpoint, err)
		return
	}
	log.WithField("endpoint", s.endpoint).Info("Connected to beacon node via RPC")
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
