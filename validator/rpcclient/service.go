// Package rpcclient defines a gRPC connection to a beacon node.
package rpcclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"

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
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	providerURL, err := url.Parse(s.endpoint)
	if err != nil {
		log.Fatalf("Unable to parse beacon RPC provider endpoint url: %v", err)
	}
	conn, err := grpc.Dial(fmt.Sprintf("[%s]:%s", providerURL.Hostname(), providerURL.Port()), server)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", s.endpoint, err)
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

// Status returns error if there is no connection to the beacon chain RPC.
func (s *Service) Status() error {
	if s.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}

// BeaconServiceClient initializes a new beacon gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (s *Service) BeaconServiceClient() pb.BeaconServiceClient {
	return pb.NewBeaconServiceClient(s.conn)
}

// ProposerServiceClient initializes a new proposer gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (s *Service) ProposerServiceClient() pb.ProposerServiceClient {
	return pb.NewProposerServiceClient(s.conn)
}

// AttesterServiceClient initializes a new attester gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (s *Service) AttesterServiceClient() pb.AttesterServiceClient {
	return pb.NewAttesterServiceClient(s.conn)
}

// ValidatorServiceClient initializes a new validator gRPC service using
// an underlying connection object.
// This wrapper is important because the underlying gRPC connection is
// only defined after the service .Start() function is called.
func (s *Service) ValidatorServiceClient() pb.ValidatorServiceClient {
	return pb.NewValidatorServiceClient(s.conn)
}
