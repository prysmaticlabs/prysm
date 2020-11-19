package remote

import (
	"context"
	"fmt"
	"strings"
	"time"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	ErrSlasherUnavailable = errors.New("slasher server is unavailable")
	log                   = logrus.WithField("prefix", "remote-slashing-protection")
)

// Config for the slashing protection service.
type Config struct {
	SlasherEndpoint            string
	CertFlag                   string
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
	GrpcRetryDelay             time.Duration
	GrpcHeadersFlag            string
}

// Service for remote slashing protection.
type Service struct {
	ctx             context.Context
	slasherEndpoint string
	slasherClient   slashpb.SlasherClient
	conn            *grpc.ClientConn
	opts            []grpc.DialOption
}

// NewService for remote slashing protection.
func NewService(ctx context.Context, config *Config) (*Service, error) {
	var dialOpt grpc.DialOption
	if config.CertFlag != "" {
		creds, err := credentials.NewClientTLSFromFile(config.CertFlag, "")
		if err != nil {
			return nil, errors.Wrap(err, "could not get valid slasher credentials")
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure slasher gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	md := make(metadata.MD)
	for _, hdr := range strings.Split(config.GrpcHeadersFlag, ",") {
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
			grpc_retry.WithMax(config.GrpcRetriesFlag),
			grpc_retry.WithBackoff(grpc_retry.BackoffLinear(config.GrpcRetryDelay)),
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
	return &Service{
		ctx:             ctx,
		slasherEndpoint: config.SlasherEndpoint,
		opts:            opts,
	}, nil
}

// Start the remote protector service.
func (rp *Service) Start() {
	conn, err := grpc.DialContext(rp.ctx, rp.slasherEndpoint, rp.opts...)
	if err != nil {
		log.Errorf("Could not dial slasher endpoint: %s", rp.slasherEndpoint)
		return
	}
	rp.slasherClient = slashpb.NewSlasherClient(conn)
	log.Debug("Successfully started slasher gRPC connection")
}

// Stop the remote protector service.
func (rp *Service) Stop() error {
	log.Info("Stopping remote slashing protector")
	if rp.conn != nil {
		return rp.conn.Close()
	}
	return nil
}

// Status checks if the connection to slasher server is ready,
// returns error otherwise.
func (rp *Service) Status() error {
	if rp.conn == nil {
		return errors.New("no connection to slasher RPC")
	}
	if rp.conn.GetState() != connectivity.Ready {
		return fmt.Errorf(
			"cannot connect to slasher server at: %v connection status: %v",
			rp.slasherEndpoint,
			rp.conn.GetState(),
		)
	}
	return nil
}

func parseSlasherError(err error) error {
	log.WithError(err).Errorf("Received an error from slasher for remote slashing protection")
	switch status.Code(err) {
	case codes.ResourceExhausted:
		return ErrSlasherUnavailable
	case codes.Canceled:
		return ErrSlasherUnavailable
	case codes.Unavailable:
		return ErrSlasherUnavailable
	}
	return errors.Wrap(err, "remote slashing block protection returned an error")
}
