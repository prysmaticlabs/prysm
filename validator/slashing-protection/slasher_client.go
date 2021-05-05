package slashingprotection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	ethsl "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

// Service represents a service to manage the validator
// ￿slashing protection.
type Service struct {
	cfg           *Config
	ctx           context.Context
	cancel        context.CancelFunc
	conn          *grpc.ClientConn
	grpcHeaders   []string
	slasherClient ethsl.SlasherClient
}

// Config for the validator service.
type Config struct {
	Endpoint                   string
	CertFlag                   string
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
	GrpcRetryDelay             time.Duration
	GrpcHeadersFlag            string
}

// NewService creates a new validator service for the service
// registry.
func NewService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
		grpcHeaders: strings.Split(cfg.GrpcHeadersFlag, ","),
	}, nil
}

// Start the slasher protection service and grpc client.
func (s *Service) Start() {
	if s.cfg.Endpoint != "" {
		s.slasherClient = s.startSlasherClient()
	}
}

func (s *Service) startSlasherClient() ethsl.SlasherClient {
	var dialOpt grpc.DialOption

	if s.cfg.CertFlag != "" {
		creds, err := credentials.NewClientTLSFromFile(s.cfg.CertFlag, "")
		if err != nil {
			log.Errorf("Could not get valid slasher credentials: %v", err)
			return nil
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure slasher gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	s.ctx = grpcutils.AppendHeaders(s.ctx, s.grpcHeaders)

	opts := []grpc.DialOption{
		dialOpt,
		grpc.WithDefaultCallOptions(
			grpc_retry.WithMax(s.cfg.GrpcRetriesFlag),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(s.cfg.GrpcRetryDelay)),
		),
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
			grpc_retry.StreamClientInterceptor(),
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
			grpc_retry.UnaryClientInterceptor(),
			grpcutils.LogRequests,
		)),
	}
	conn, err := grpc.DialContext(s.ctx, s.cfg.Endpoint, opts...)
	if err != nil {
		log.Errorf("Could not dial slasher endpoint: %s, %v", s.cfg.Endpoint, err)
		return nil
	}
	log.Debug("Successfully started slasher gRPC connection")
	s.conn = conn
	return ethsl.NewSlasherClient(s.conn)

}

// Stop the validator service.
func (s *Service) Stop() error {
	s.cancel()
	log.Info("Stopping slashing protection service")
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Status checks if the connection to slasher server is ready,
// returns error otherwise.
func (s *Service) Status() error {
	if s.conn == nil {
		return errors.New("no connection to slasher RPC")
	}
	if s.conn.GetState() != connectivity.Ready {
		return fmt.Errorf("can`t connect to slasher server at: %v connection status: %v ", s.cfg.Endpoint, s.conn.GetState())
	}
	return nil
}
