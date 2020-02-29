package client

import (
	"context"

	"github.com/dgraph-io/ristretto"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/db"
	"github.com/prysmaticlabs/prysm/validator/keymanager"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "validator")

// ValidatorService represents a service to manage the validator client
// routine.
type ValidatorService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	validator            Validator
	graffiti             []byte
	conn                 *grpc.ClientConn
	endpoint             string
	withCert             string
	dataDir              string
	keyManager           keymanager.KeyManager
	logValidatorBalances bool
	emitAccountMetrics   bool
	maxCallRecvMsgSize   int
	grpcRetries          uint
}

// Config for the validator service.
type Config struct {
	Endpoint                   string
	DataDir                    string
	CertFlag                   string
	GraffitiFlag               string
	KeyManager                 keymanager.KeyManager
	LogValidatorBalances       bool
	EmitAccountMetrics         bool
	GrpcMaxCallRecvMsgSizeFlag int
	GrpcRetriesFlag            uint
}

// NewValidatorService creates a new validator service for the service
// registry.
func NewValidatorService(ctx context.Context, cfg *Config) (*ValidatorService, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &ValidatorService{
		ctx:                  ctx,
		cancel:               cancel,
		endpoint:             cfg.Endpoint,
		withCert:             cfg.CertFlag,
		dataDir:              cfg.DataDir,
		graffiti:             []byte(cfg.GraffitiFlag),
		keyManager:           cfg.KeyManager,
		logValidatorBalances: cfg.LogValidatorBalances,
		emitAccountMetrics:   cfg.EmitAccountMetrics,
		maxCallRecvMsgSize:   cfg.GrpcMaxCallRecvMsgSizeFlag,
		grpcRetries:          cfg.GrpcRetriesFlag,
	}, nil
}

// Start the validator service. Launches the main go routine for the validator
// client.
func (v *ValidatorService) Start() {
	var dialOpt grpc.DialOption
	var maxCallRecvMsgSize int

	if v.withCert != "" {
		creds, err := credentials.NewClientTLSFromFile(v.withCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			return
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}

	if v.maxCallRecvMsgSize != 0 {
		maxCallRecvMsgSize = v.maxCallRecvMsgSize
	} else {
		maxCallRecvMsgSize = 10 * 5 << 20 // Default 50Mb
	}

	opts := []grpc.DialOption{
		dialOpt,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxCallRecvMsgSize),
			grpc_retry.WithMax(v.grpcRetries),
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
			logDebugRequestInfoUnaryInterceptor,
		)),
	}
	conn, err := grpc.DialContext(v.ctx, v.endpoint, opts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", v.endpoint, err)
		return
	}
	log.Info("Successfully started gRPC connection")

	pubkeys, err := v.keyManager.FetchValidatingKeys()
	if err != nil {
		log.Errorf("Could not get validating keys: %v", err)
		return
	}

	valDB, err := db.NewKVStore(v.dataDir, pubkeys)
	if err != nil {
		log.Errorf("Could not initialize db: %v", err)
		return
	}

	v.conn = conn
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1280, // number of keys to track.
		MaxCost:     128,  // maximum cost of cache, 1 item = 1 cost.
		BufferItems: 64,   // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}

	v.validator = &validator{
		db:                   valDB,
		validatorClient:      ethpb.NewBeaconNodeValidatorClient(v.conn),
		beaconClient:         ethpb.NewBeaconChainClient(v.conn),
		node:                 ethpb.NewNodeClient(v.conn),
		keyManager:           v.keyManager,
		graffiti:             v.graffiti,
		logValidatorBalances: v.logValidatorBalances,
		emitAccountMetrics:   v.emitAccountMetrics,
		prevBalance:          make(map[[48]byte]uint64),
		attLogs:              make(map[[32]byte]*attSubmitted),
		domainDataCache:      cache,
	}
	go run(v.ctx, v.validator)
}

// Stop the validator service.
func (v *ValidatorService) Stop() error {
	v.cancel()
	log.Info("Stopping service")
	if v.conn != nil {
		return v.conn.Close()
	}
	return nil
}

// Status ...
//
// WIP - not done.
func (v *ValidatorService) Status() error {
	if v.conn == nil {
		return errors.New("no connection to beacon RPC")
	}
	return nil
}
