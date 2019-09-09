// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache/depositcache"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var log logrus.FieldLogger

func init() {
	log = logrus.WithField("prefix", "rpc")
}

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	beaconDB            db.Database
	stateFeedListener   blockchain.ChainFeeds
	headFetcher         blockchain.HeadFetcher
	attestationReceiver blockchain.AttestationReceiver
	blockReceiver       blockchain.BlockReceiver
	powChainService     powchain.Chain
	mockEth1Votes       bool
	attestationsPool    operations.Pool
	operationsHandler   operations.Handler
	syncService         sync.Checker
	port                string
	listener            net.Listener
	withCert            string
	withKey             string
	grpcServer          *grpc.Server
	canonicalStateChan  chan *pbp2p.BeaconState
	incomingAttestation chan *ethpb.Attestation
	credentialError     error
	p2p                 p2p.Broadcaster
	depositCache        *depositcache.DepositCache
}

// Config options for the beacon node RPC server.
type Config struct {
	Port                string
	CertFlag            string
	KeyFlag             string
	BeaconDB            db.Database
	StateFeedListener   blockchain.ChainFeeds
	HeadFetcher         blockchain.HeadFetcher
	AttestationReceiver blockchain.AttestationReceiver
	BlockReceiver       blockchain.BlockReceiver
	POWChainService     powchain.Chain
	MockEth1Votes       bool
	OperationsHandler   operations.Handler
	AttestationsPool    operations.Pool
	SyncService         sync.Checker
	Broadcaster         p2p.Broadcaster
	DepositCache        *depositcache.DepositCache
}

// NewService instantiates a new RPC service instance that will
// be registered into a running beacon node.
func NewService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                 ctx,
		cancel:              cancel,
		beaconDB:            cfg.BeaconDB,
		stateFeedListener:   cfg.StateFeedListener,
		headFetcher:         cfg.HeadFetcher,
		attestationReceiver: cfg.AttestationReceiver,
		blockReceiver:       cfg.BlockReceiver,
		p2p:                 cfg.Broadcaster,
		powChainService:     cfg.POWChainService,
		mockEth1Votes:       cfg.MockEth1Votes,
		attestationsPool:    cfg.AttestationsPool,
		operationsHandler:   cfg.OperationsHandler,
		syncService:         cfg.SyncService,
		port:                cfg.Port,
		withCert:            cfg.CertFlag,
		withKey:             cfg.KeyFlag,
		depositCache:        cfg.DepositCache,
		canonicalStateChan:  make(chan *pbp2p.BeaconState, params.BeaconConfig().DefaultBufferSize),
		incomingAttestation: make(chan *ethpb.Attestation, params.BeaconConfig().DefaultBufferSize),
	}
}

// Start the gRPC server.
func (s *Service) Start() {
	log.Info("Starting service")
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

	beaconServer := &BeaconServer{
		beaconDB:            s.beaconDB,
		ctx:                 s.ctx,
		chainStartFetcher:   s.powChainService,
		eth1InfoFetcher:     s.powChainService,
		headFetcher:         s.headFetcher,
		stateFeedListener:   s.stateFeedListener,
		incomingAttestation: s.incomingAttestation,
		canonicalStateChan:  s.canonicalStateChan,
		chainStartChan:      make(chan time.Time, 1),
	}
	proposerServer := &ProposerServer{
		beaconDB:           s.beaconDB,
		headFetcher:        s.headFetcher,
		blockReceiver:      s.blockReceiver,
		chainStartFetcher:  s.powChainService,
		eth1InfoFetcher:    s.powChainService,
		eth1BlockFetcher:   s.powChainService,
		mockEth1Votes:      s.mockEth1Votes,
		pool:               s.attestationsPool,
		canonicalStateChan: s.canonicalStateChan,
		depositCache:       s.depositCache,
	}
	attesterServer := &AttesterServer{
		p2p:               s.p2p,
		beaconDB:          s.beaconDB,
		operationsHandler: s.operationsHandler,
		attReceiver:       s.attestationReceiver,
		headFetcher:       s.headFetcher,
		depositCache:      cache.NewAttestationCache(),
	}
	validatorServer := &ValidatorServer{
		ctx:                s.ctx,
		beaconDB:           s.beaconDB,
		headFetcher:        s.headFetcher,
		canonicalStateChan: s.canonicalStateChan,
		blockFetcher:       s.powChainService,
		chainStartFetcher:  s.powChainService,
		depositCache:       s.depositCache,
	}
	nodeServer := &NodeServer{
		beaconDB:    s.beaconDB,
		server:      s.grpcServer,
		syncChecker: s.syncService,
	}
	beaconChainServer := &BeaconChainServer{
		beaconDB:    s.beaconDB,
		pool:        s.attestationsPool,
		headFetcher: s.headFetcher,
	}
	pb.RegisterBeaconServiceServer(s.grpcServer, beaconServer)
	pb.RegisterProposerServiceServer(s.grpcServer, proposerServer)
	pb.RegisterAttesterServiceServer(s.grpcServer, attesterServer)
	pb.RegisterValidatorServiceServer(s.grpcServer, validatorServer)
	ethpb.RegisterNodeServer(s.grpcServer, nodeServer)
	ethpb.RegisterBeaconChainServer(s.grpcServer, beaconChainServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		for s.syncService.Status() != nil {
			time.Sleep(time.Second * params.BeaconConfig().RPCSyncCheck)
		}
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
	s.cancel()
	if s.listener != nil {
		s.grpcServer.GracefulStop()
		log.Debug("Initiated graceful stop of gRPC server")
	}
	return nil
}

// Status returns nil or credentialError
func (s *Service) Status() error {
	if s.credentialError != nil {
		return s.credentialError
	}
	return nil
}
