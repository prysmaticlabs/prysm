package rpcclient

import (
	"context"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.WithField("prefix", "beaconclient")

// Service hi.
type Service struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *grpc.ClientConn
}

// NewRPCClient hi.
func NewRPCClient(ctx context.Context) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start hi.
func (s *Service) Start() {
	conn, err := grpc.Dial("localhost:8080")
	if err != nil {
		panic(err)
	}
	s.conn = conn
}

// Stop hi.
func (s *Service) Stop() error {
	defer s.conn.Close()
	return nil
}

// BeaconServiceClient hi.
func (s *Service) BeaconServiceClient() pb.BeaconServiceClient {
	return pb.NewBeaconServiceClient(s.conn)
}
