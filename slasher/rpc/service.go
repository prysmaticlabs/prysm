// Package rpc defines an implementation of a gRPC slasher service,
// providing endpoints for determining whether or not a block/attestation
// is slashable based on slasher's evidence.
package rpc

import (
	"context"
	"fmt"
	"net"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/prysmaticlabs/prysm/slasher/beaconclient"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Service defines a server implementation of the gRPC Slasher service,
// providing RPC endpoints for retrieving slashing proofs for malicious validators.
type Service struct {
	ctx             context.Context
	cancel          context.CancelFunc
	host            string
	port            string
	detector        *detection.Service
	listener        net.Listener
	grpcServer      *grpc.Server
	slasherDB       db.Database
	withCert        string
	withKey         string
	credentialError error
	beaconclient    *beaconclient.Service
}

// Config options for the slasher node RPC server.
type Config struct {
	Host         string
	Port         string
	CertFlag     string
	KeyFlag      string
	Detector     *detection.Service
	SlasherDB    db.Database
	BeaconClient *beaconclient.Service
}

var log = logrus.WithField("prefix", "rpc")

// NewService instantiates a new RPC service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:          ctx,
		cancel:       cancel,
		host:         cfg.Host,
		port:         cfg.Port,
		detector:     cfg.Detector,
		slasherDB:    cfg.SlasherDB,
		withCert:     cfg.CertFlag,
		withKey:      cfg.KeyFlag,
		beaconclient: cfg.BeaconClient,
	}
}

// Start the gRPC service.
func (s *Service) Start() {
	address := fmt.Sprintf("%s:%s", s.host, s.port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("Could not listen to port in Start() %s: %v", address, err)
	}
	s.listener = lis
	log.WithField("address", address).Info("RPC-API listening on port")

	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.StreamInterceptor(middleware.ChainStreamServer(
			recovery.StreamServerInterceptor(
				recovery.WithRecoveryHandlerContext(traceutil.RecoveryHandlerFunc),
			),
			grpc_prometheus.StreamServerInterceptor,
			grpc_opentracing.StreamServerInterceptor(),
		)),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandlerContext(traceutil.RecoveryHandlerFunc),
			),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_opentracing.UnaryServerInterceptor(),
		)),
	}
	grpc_prometheus.EnableHandlingTimeHistogram()
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
		log.Warn("You are using an insecure gRPC server. If you are running your slasher and " +
			"validator on the same machines, you can ignore this message. If you want to know " +
			"how to enable secure connections, see: https://docs.prylabs.network/docs/prysm-usage/secure-grpc")
	}
	s.grpcServer = grpc.NewServer(opts...)

	slasherServer := &Server{
		ctx:          s.ctx,
		detector:     s.detector,
		slasherDB:    s.slasherDB,
		beaconClient: s.beaconclient,
	}
	slashpb.RegisterSlasherServer(s.grpcServer, slasherServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		if s.listener == nil {
			return
		}
		if err := s.grpcServer.Serve(s.listener); err != nil {
			log.Errorf("Could not serve gRPC: %v", err)
		}
	}()
}

// Stop the service.
func (s *Service) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	return nil
}

// Status returns nil if slasher is ready to receive attestations and
// blocks from clients for slashing detection.
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	if bs := s.beaconclient.Status(); bs != nil {
		return bs
	}
	if ds := s.detector.Status(); ds != nil {
		return ds
	}
	return nil
}
