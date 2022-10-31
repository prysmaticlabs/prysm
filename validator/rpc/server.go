package rpc

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/io/logs"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpbservice "github.com/prysmaticlabs/prysm/v3/proto/eth/service"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	validatorpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1/validator-client"
	"github.com/prysmaticlabs/prysm/v3/validator/accounts/wallet"
	"github.com/prysmaticlabs/prysm/v3/validator/client"
	"github.com/prysmaticlabs/prysm/v3/validator/db"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Config options for the gRPC server.
type Config struct {
	ValDB                    db.Database
	GenesisFetcher           client.GenesisFetcher
	SyncChecker              client.SyncChecker
	Wallet                   *wallet.Wallet
	WalletInitializedFeed    *event.Feed
	ValidatorService         *client.ValidatorService
	WalletDir                string
	KeyFlag                  string
	NodeGatewayEndpoint      string
	ClientWithCert           string
	Host                     string
	Port                     string
	CertFlag                 string
	ValidatorMonitoringHost  string
	BeaconClientEndpoint     string
	ValidatorGatewayHost     string
	ClientGrpcHeaders        []string
	ClientMaxCallRecvMsgSize int
	ClientGrpcRetries        uint
	ValidatorMonitoringPort  int
	ClientGrpcRetryDelay     time.Duration
	ValidatorGatewayPort     int
}

// Server defining a gRPC server for the remote signer API.
type Server struct {
	listener                  net.Listener
	genesisFetcher            client.GenesisFetcher
	beaconChainClient         ethpb.BeaconChainClient
	beaconNodeClient          ethpb.NodeClient
	beaconNodeValidatorClient ethpb.BeaconNodeValidatorClient
	beaconNodeHealthClient    ethpb.HealthClient
	valDB                     db.Database
	ctx                       context.Context
	logsStreamer              logs.Streamer
	syncChecker               client.SyncChecker
	credentialError           error
	cancel                    context.CancelFunc
	grpcServer                *grpc.Server
	wallet                    *wallet.Wallet
	validatorService          *client.ValidatorService
	walletInitializedFeed     *event.Feed
	host                      string
	validatorMonitoringHost   string
	withCert                  string
	withKey                   string
	nodeGatewayEndpoint       string
	port                      string
	walletDir                 string
	clientWithCert            string
	beaconClientEndpoint      string
	validatorGatewayHost      string
	jwtSecret                 []byte
	clientGrpcHeaders         []string
	streamLogsBufferSize      int
	clientGrpcRetryDelay      time.Duration
	clientMaxCallRecvMsgSize  int
	clientGrpcRetries         uint
	validatorMonitoringPort   int
	validatorGatewayPort      int
	walletInitialized         bool
}

// NewServer instantiates a new gRPC server.
func NewServer(ctx context.Context, cfg *Config) *Server {
	ctx, cancel := context.WithCancel(ctx)
	return &Server{
		ctx:                      ctx,
		cancel:                   cancel,
		logsStreamer:             logs.NewStreamServer(),
		streamLogsBufferSize:     1000, // Enough to handle most bursts of logs in the validator client.
		host:                     cfg.Host,
		port:                     cfg.Port,
		withCert:                 cfg.CertFlag,
		withKey:                  cfg.KeyFlag,
		beaconClientEndpoint:     cfg.BeaconClientEndpoint,
		clientMaxCallRecvMsgSize: cfg.ClientMaxCallRecvMsgSize,
		clientGrpcRetries:        cfg.ClientGrpcRetries,
		clientGrpcRetryDelay:     cfg.ClientGrpcRetryDelay,
		clientGrpcHeaders:        cfg.ClientGrpcHeaders,
		clientWithCert:           cfg.ClientWithCert,
		valDB:                    cfg.ValDB,
		validatorService:         cfg.ValidatorService,
		syncChecker:              cfg.SyncChecker,
		genesisFetcher:           cfg.GenesisFetcher,
		walletDir:                cfg.WalletDir,
		walletInitializedFeed:    cfg.WalletInitializedFeed,
		walletInitialized:        cfg.Wallet != nil,
		wallet:                   cfg.Wallet,
		nodeGatewayEndpoint:      cfg.NodeGatewayEndpoint,
		validatorMonitoringHost:  cfg.ValidatorMonitoringHost,
		validatorMonitoringPort:  cfg.ValidatorMonitoringPort,
		validatorGatewayHost:     cfg.ValidatorGatewayHost,
		validatorGatewayPort:     cfg.ValidatorGatewayPort,
	}
}

// Start the gRPC server.
func (s *Server) Start() {
	// Setup the gRPC server options and TLS configuration.
	address := fmt.Sprintf("%s:%s", s.host, s.port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.WithError(err).Errorf("Could not listen to port in Start() %s", address)
	}
	s.listener = lis

	// Register interceptors for metrics gathering as well as our
	// own, custom JWT unary interceptor.
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandlerContext(tracing.RecoveryHandlerFunc),
			),
			grpcprometheus.UnaryServerInterceptor,
			grpcopentracing.UnaryServerInterceptor(),
			s.JWTInterceptor(),
		)),
	}
	grpcprometheus.EnableHandlingTimeHistogram()

	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.WithError(err).Fatal("Could not load TLS keys")
		}
		opts = append(opts, grpc.Creds(creds))
		log.WithFields(logrus.Fields{
			"crt-path": s.withCert,
			"key-path": s.withKey,
		}).Info("Loaded TLS certificates")
	}
	s.grpcServer = grpc.NewServer(opts...)

	// Register a gRPC client to the beacon node.
	if err := s.registerBeaconClient(); err != nil {
		log.WithError(err).Fatal("Could not register beacon chain gRPC client")
	}

	// Register services available for the gRPC server.
	reflection.Register(s.grpcServer)
	validatorpb.RegisterAuthServer(s.grpcServer, s)
	validatorpb.RegisterWalletServer(s.grpcServer, s)
	validatorpb.RegisterHealthServer(s.grpcServer, s)
	validatorpb.RegisterBeaconServer(s.grpcServer, s)
	validatorpb.RegisterAccountsServer(s.grpcServer, s)
	ethpbservice.RegisterKeyManagementServer(s.grpcServer, s)
	validatorpb.RegisterSlashingProtectionServer(s.grpcServer, s)

	go func() {
		if s.listener != nil {
			if err := s.grpcServer.Serve(s.listener); err != nil {
				log.WithError(err).Error("Could not serve")
			}
		}
	}()
	log.WithField("address", address).Info("gRPC server listening on address")
	if s.walletDir != "" {
		token, err := s.initializeAuthToken(s.walletDir)
		if err != nil {
			log.WithError(err).Error("Could not initialize web auth token")
			return
		}
		validatorWebAddr := fmt.Sprintf("%s:%d", s.validatorGatewayHost, s.validatorGatewayPort)
		authTokenPath := filepath.Join(s.walletDir, authTokenFileName)
		logValidatorWebAuth(validatorWebAddr, token, authTokenPath)
		go s.refreshAuthTokenFromFileChanges(s.ctx, authTokenPath)
	}
}

// Stop the gRPC server.
func (s *Server) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of server")
	}
	return nil
}

// Status returns nil or credentialError.
func (s *Server) Status() error {
	return s.credentialError
}
