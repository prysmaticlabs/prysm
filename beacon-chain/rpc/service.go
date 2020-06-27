// Package rpc defines a gRPC server implementing the eth2 API as needed
// by validator clients and consumers of chain data.
package rpc

import (
	"context"
	"fmt"
	"net"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/attestations"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/slashings"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/voluntaryexits"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/beacon"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/debug"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/node"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/validator"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc")
}

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx                     context.Context
	cancel                  context.CancelFunc
	beaconDB                db.HeadAccessDatabase
	headFetcher             blockchain.HeadFetcher
	forkFetcher             blockchain.ForkFetcher
	finalizationFetcher     blockchain.FinalizationFetcher
	participationFetcher    blockchain.ParticipationFetcher
	genesisTimeFetcher      blockchain.TimeFetcher
	genesisFetcher          blockchain.GenesisFetcher
	attestationReceiver     blockchain.AttestationReceiver
	blockReceiver           blockchain.BlockReceiver
	powChainService         powchain.Chain
	chainStartFetcher       powchain.ChainStartFetcher
	mockEth1Votes           bool
	enableDebugRPCEndpoints bool
	attestationsPool        attestations.Pool
	exitPool                *voluntaryexits.Pool
	slashingsPool           *slashings.Pool
	syncService             sync.Checker
	host                    string
	port                    string
	listener                net.Listener
	withCert                string
	withKey                 string
	grpcServer              *grpc.Server
	canonicalStateChan      chan *pbp2p.BeaconState
	incomingAttestation     chan *ethpb.Attestation
	credentialError         error
	p2p                     p2p.Broadcaster
	peersFetcher            p2p.PeersProvider
	peerManager             p2p.PeerManager
	depositFetcher          depositcache.DepositFetcher
	pendingDepositFetcher   depositcache.PendingDepositsFetcher
	stateNotifier           statefeed.Notifier
	blockNotifier           blockfeed.Notifier
	operationNotifier       opfeed.Notifier
	slasherConn             *grpc.ClientConn
	slasherProvider         string
	slasherCert             string
	slasherCredentialError  error
	slasherClient           slashpb.SlasherClient
	stateGen                *stategen.State
	connectedRPCClients     map[net.Addr]bool
}

// Config options for the beacon node RPC server.
type Config struct {
	Host                    string
	Port                    string
	CertFlag                string
	KeyFlag                 string
	BeaconDB                db.HeadAccessDatabase
	HeadFetcher             blockchain.HeadFetcher
	ForkFetcher             blockchain.ForkFetcher
	FinalizationFetcher     blockchain.FinalizationFetcher
	ParticipationFetcher    blockchain.ParticipationFetcher
	AttestationReceiver     blockchain.AttestationReceiver
	BlockReceiver           blockchain.BlockReceiver
	POWChainService         powchain.Chain
	ChainStartFetcher       powchain.ChainStartFetcher
	GenesisTimeFetcher      blockchain.TimeFetcher
	GenesisFetcher          blockchain.GenesisFetcher
	EnableDebugRPCEndpoints bool
	MockEth1Votes           bool
	AttestationsPool        attestations.Pool
	ExitPool                *voluntaryexits.Pool
	SlashingsPool           *slashings.Pool
	SyncService             sync.Checker
	Broadcaster             p2p.Broadcaster
	PeersFetcher            p2p.PeersProvider
	PeerManager             p2p.PeerManager
	DepositFetcher          depositcache.DepositFetcher
	PendingDepositFetcher   depositcache.PendingDepositsFetcher
	SlasherProvider         string
	SlasherCert             string
	StateNotifier           statefeed.Notifier
	BlockNotifier           blockfeed.Notifier
	OperationNotifier       opfeed.Notifier
	StateGen                *stategen.State
}

// NewService instantiates a new RPC service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                     ctx,
		cancel:                  cancel,
		beaconDB:                cfg.BeaconDB,
		headFetcher:             cfg.HeadFetcher,
		forkFetcher:             cfg.ForkFetcher,
		finalizationFetcher:     cfg.FinalizationFetcher,
		participationFetcher:    cfg.ParticipationFetcher,
		genesisTimeFetcher:      cfg.GenesisTimeFetcher,
		genesisFetcher:          cfg.GenesisFetcher,
		attestationReceiver:     cfg.AttestationReceiver,
		blockReceiver:           cfg.BlockReceiver,
		p2p:                     cfg.Broadcaster,
		peersFetcher:            cfg.PeersFetcher,
		peerManager:             cfg.PeerManager,
		powChainService:         cfg.POWChainService,
		chainStartFetcher:       cfg.ChainStartFetcher,
		mockEth1Votes:           cfg.MockEth1Votes,
		attestationsPool:        cfg.AttestationsPool,
		exitPool:                cfg.ExitPool,
		slashingsPool:           cfg.SlashingsPool,
		syncService:             cfg.SyncService,
		host:                    cfg.Host,
		port:                    cfg.Port,
		withCert:                cfg.CertFlag,
		withKey:                 cfg.KeyFlag,
		depositFetcher:          cfg.DepositFetcher,
		pendingDepositFetcher:   cfg.PendingDepositFetcher,
		canonicalStateChan:      make(chan *pbp2p.BeaconState, params.BeaconConfig().DefaultBufferSize),
		incomingAttestation:     make(chan *ethpb.Attestation, params.BeaconConfig().DefaultBufferSize),
		stateNotifier:           cfg.StateNotifier,
		blockNotifier:           cfg.BlockNotifier,
		operationNotifier:       cfg.OperationNotifier,
		slasherProvider:         cfg.SlasherProvider,
		slasherCert:             cfg.SlasherCert,
		stateGen:                cfg.StateGen,
		enableDebugRPCEndpoints: cfg.EnableDebugRPCEndpoints,
		connectedRPCClients:     make(map[net.Addr]bool),
	}
}

// Start the gRPC server.
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
			s.validatorStreamConnectionInterceptor,
		)),
		grpc.UnaryInterceptor(middleware.ChainUnaryServer(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandlerContext(traceutil.RecoveryHandlerFunc),
			),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_opentracing.UnaryServerInterceptor(),
			s.validatorUnaryConnectionInterceptor,
		)),
	}
	grpc_prometheus.EnableHandlingTimeHistogram()
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("Could not load TLS keys: %s", err)
			s.credentialError = err
		}
		opts = append(opts, grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC server. If you are running your beacon node and " +
			"validator on the same machines, you can ignore this message. If you want to know " +
			"how to enable secure connections, see: https://docs.prylabs.network/docs/prysm-usage/secure-grpc")
	}
	s.grpcServer = grpc.NewServer(opts...)

	validatorServer := &validator.Server{
		Ctx:                    s.ctx,
		BeaconDB:               s.beaconDB,
		AttestationCache:       cache.NewAttestationCache(),
		AttPool:                s.attestationsPool,
		ExitPool:               s.exitPool,
		HeadFetcher:            s.headFetcher,
		ForkFetcher:            s.forkFetcher,
		FinalizationFetcher:    s.finalizationFetcher,
		GenesisTimeFetcher:     s.genesisTimeFetcher,
		CanonicalStateChan:     s.canonicalStateChan,
		BlockFetcher:           s.powChainService,
		DepositFetcher:         s.depositFetcher,
		ChainStartFetcher:      s.chainStartFetcher,
		Eth1InfoFetcher:        s.powChainService,
		SyncChecker:            s.syncService,
		StateNotifier:          s.stateNotifier,
		BlockNotifier:          s.blockNotifier,
		OperationNotifier:      s.operationNotifier,
		P2P:                    s.p2p,
		BlockReceiver:          s.blockReceiver,
		MockEth1Votes:          s.mockEth1Votes,
		Eth1BlockFetcher:       s.powChainService,
		PendingDepositsFetcher: s.pendingDepositFetcher,
		SlashingsPool:          s.slashingsPool,
		StateGen:               s.stateGen,
	}
	nodeServer := &node.Server{
		BeaconDB:           s.beaconDB,
		Server:             s.grpcServer,
		SyncChecker:        s.syncService,
		GenesisTimeFetcher: s.genesisTimeFetcher,
		PeersFetcher:       s.peersFetcher,
		PeerManager:        s.peerManager,
		GenesisFetcher:     s.genesisFetcher,
	}
	beaconChainServer := &beacon.Server{
		Ctx:                         s.ctx,
		BeaconDB:                    s.beaconDB,
		AttestationsPool:            s.attestationsPool,
		SlashingsPool:               s.slashingsPool,
		HeadFetcher:                 s.headFetcher,
		FinalizationFetcher:         s.finalizationFetcher,
		ParticipationFetcher:        s.participationFetcher,
		ChainStartFetcher:           s.chainStartFetcher,
		DepositFetcher:              s.depositFetcher,
		BlockFetcher:                s.powChainService,
		CanonicalStateChan:          s.canonicalStateChan,
		GenesisTimeFetcher:          s.genesisTimeFetcher,
		StateNotifier:               s.stateNotifier,
		BlockNotifier:               s.blockNotifier,
		AttestationNotifier:         s.operationNotifier,
		Broadcaster:                 s.p2p,
		StateGen:                    s.stateGen,
		SyncChecker:                 s.syncService,
		ReceivedAttestationsBuffer:  make(chan *ethpb.Attestation, 100),
		CollectedAttestationsBuffer: make(chan []*ethpb.Attestation, 100),
	}
	ethpb.RegisterNodeServer(s.grpcServer, nodeServer)
	ethpb.RegisterBeaconChainServer(s.grpcServer, beaconChainServer)
	if s.enableDebugRPCEndpoints {
		log.Info("Enabled debug RPC endpoints")
		debugServer := &debug.Server{
			GenesisTimeFetcher: s.genesisTimeFetcher,
			StateGen:           s.stateGen,
			HeadFetcher:        s.headFetcher,
			PeerManager:        s.peerManager,
			PeersFetcher:       s.peersFetcher,
		}
		pbrpc.RegisterDebugServer(s.grpcServer, debugServer)
	}
	ethpb.RegisterBeaconNodeValidatorServer(s.grpcServer, validatorServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		if s.listener != nil {
			if err := s.grpcServer.Serve(s.listener); err != nil {
				log.Errorf("Could not serve gRPC: %v", err)
			}
		}
	}()
	if featureconfig.Get().EnableSlasherConnection {
		s.startSlasherClient()
	}
}

func (s *Service) startSlasherClient() {
	var dialOpt grpc.DialOption
	if s.slasherCert != "" {
		creds, err := credentials.NewClientTLSFromFile(s.slasherCert, "")
		if err != nil {
			log.Errorf("Could not get valid credentials: %v", err)
			s.slasherCredentialError = err
		}
		dialOpt = grpc.WithTransportCredentials(creds)
	} else {
		dialOpt = grpc.WithInsecure()
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection.")
	}
	slasherOpts := []grpc.DialOption{
		dialOpt,
		grpc.WithStatsHandler(&ocgrpc.ClientHandler{}),
		grpc.WithStreamInterceptor(middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(),
			grpc_prometheus.StreamClientInterceptor,
		)),
		grpc.WithUnaryInterceptor(middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(),
			grpc_prometheus.UnaryClientInterceptor,
		)),
	}
	conn, err := grpc.DialContext(s.ctx, s.slasherProvider, slasherOpts...)
	if err != nil {
		log.Errorf("Could not dial endpoint: %s, %v", s.slasherProvider, err)
		return
	}
	log.Info("Successfully started hash slinging slasher©️ gRPC connection")
	s.slasherConn = conn
	s.slasherClient = slashpb.NewSlasherClient(s.slasherConn)
}

// Stop the service.
func (s *Service) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	if s.slasherConn != nil {
		if err := s.slasherConn.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Status returns nil or credentialError
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	if s.slasherCredentialError != nil {
		return s.slasherCredentialError
	}
	return nil
}

// Stream interceptor for new validator client connections to the beacon node.
func (s *Service) validatorStreamConnectionInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	_ *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	s.logNewClientConnection(ss.Context())
	return handler(srv, ss)
}

// Unary interceptor for new validator client connections to the beacon node.
func (s *Service) validatorUnaryConnectionInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	s.logNewClientConnection(ctx)
	return handler(ctx, req)
}

func (s *Service) logNewClientConnection(ctx context.Context) {
	if featureconfig.Get().DisableGRPCConnectionLogs {
		return
	}
	if clientInfo, ok := peer.FromContext(ctx); ok {
		// Check if we have not yet observed this grpc client connection
		// in the running beacon node.
		if !s.connectedRPCClients[clientInfo.Addr] {
			log.WithFields(logrus.Fields{
				"addr": clientInfo.Addr.String(),
			}).Infof("New gRPC client connected to beacon node")
			s.connectedRPCClients[clientInfo.Addr] = true
		}
	}
}
