// Package slasher defines the service used to retrieve slashings proofs.
package slasher

import (
	"fmt"
	"net"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/rpc"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "slasherRPC")
}

// Service defining an RPC server for the slasher service.
type Service struct {
	slasherDb       *db.Store
	grpcServer      *grpc.Server
	port            string
	withCert        string
	withKey         string
	listener        net.Listener
	credentialError error
	failStatus      error
}

// Config options for the slasher server.
type Config struct {
	Port      string
	CertFlag  string
	KeyFlag   string
	SlasherDb *db.Store
}

// NewRPCService creates a new instance of a struct implementing the SlasherService
// interface.
func NewRPCService(cfg *Config) *Service {
	return &Service{
		slasherDb: cfg.SlasherDb,
		port:      cfg.Port,
	}
}

// Start the gRPC server.
func (s *Service) Start() {
	log.Info("Starting service on port: %v", s.port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		log.Errorf("Could not listen to port in Start() :%s: %v", s.port, err)
	}
	s.listener = lis
	log.WithField("port", s.port).Info("Listening on port")

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.StreamInterceptor(middleware.ChainStreamServer(
			recovery.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
		)),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
		)),
	}
	// TODO(#791): Utilize a certificate for secure connections
	// between beacon nodes and validator clients.
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("Could not load TLS keys: %s", err)
			s.credentialError = err
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC connection! Provide a certificate and key to connect securely")
	}
	s.grpcServer = grpc.NewServer(opts...)
	slasherServer := rpc.Server{
		SlasherDb: s.slasherDb,
	}

	ethpb.RegisterSlasherServer(s.grpcServer, &slasherServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		if s.listener != nil {
			if err := s.grpcServer.Serve(s.listener); err != nil {
				log.Errorf("Could not serve gRPC: %v", err)
			}
		}
	}()
}

// Stop the service.
func (s *Service) Stop() error {
	log.Info("Stopping service")
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	return nil
}

// Status returns nil, credentialError or fail status.
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	if s.failStatus != nil {
		return s.failStatus
	}
	return nil
}
