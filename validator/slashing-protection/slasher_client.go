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
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// Service represents a service to manage the validator
// ￿slashing protection.
type Service struct {
	ctx                context.Context
	cancel             context.CancelFunc
	conn               *grpc.ClientConn
	endpoint           string
	withCert           string
	maxCallRecvMsgSize int
	grpcRetries        uint
	grpcHeaders        []string
	slasherClient      ethsl.SlasherClient
	grpcRetryDelay     time.Duration
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

// NewSlashingProtectionService creates a new validator service for the service
// registry.
func NewSlashingProtectionService(ctx context.Context, cfg *Config) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                ctx,
		cancel:             cancel,
		endpoint:           cfg.Endpoint,
		withCert:           cfg.CertFlag,
		maxCallRecvMsgSize: cfg.GrpcMaxCallRecvMsgSizeFlag,
		grpcRetries:        cfg.GrpcRetriesFlag,
		grpcRetryDelay:     cfg.GrpcRetryDelay,
		grpcHeaders:        strings.Split(cfg.GrpcHeadersFlag, ","),
	}, nil
}

// Start the slasher protection service and grpc client.
func (s *Service) Start() {
	if s.endpoint != "" {
		s.slasherClient = s.startSlasherClient()
	}
}

func (s *Service) startSlasherClient() ethsl.SlasherClient {
	var dialOpt grpc.DialOption

	if s.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(s.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid slasher credentials: %v", err)
			return nil
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure slasher gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	md := make(metadata.MD)
	for _, hdr := range s.grpcHeaders {
		if hdr != "" {
			ss := strings.Split(hdr, "=")
			if len(ss) != 2 {
				log.Warnf("Incorrect gRPC header flag format. Skipping %v", hdr)
				continue
			}
			md.Set(ss[0], ss[1])
		}
	}

	opts := []grpc.DialOption{
		dialOpt,
		grpc.WithDefaultCallOptions(
			grpc_retry.WithMax(s.grpcRetries),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(s.grpcRetryDelay)),
			grpc.Header(&md),
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
			grpcutils.LogGRPCRequests,
		)),
	}
	conn, err := grpc.DialContext(s.ctx, s.endpoint, opts...)
	if err != nil {
		log.Errorf("Could not dial slasher endpoint: %s, %v", s.endpoint, err)
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

// Status ...
//
// WIP - not done.
func (s *Service) Status() error {
	if s.conn == nil {
		return errors.New("no connection to slasher RPC")
	}
	if s.conn.GetState() != connectivity.Ready {
		return fmt.Errorf("can`t connect to slasher server at: %v", s.endpoint)
	}
	return nil
}
