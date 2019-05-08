// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
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

type chainService interface {
	StateInitializedFeed() *event.Feed
	blockchain.BlockReceiver
	blockchain.ForkChoice
}

type operationService interface {
	PendingAttestations(ctx context.Context) ([]*pbp2p.Attestation, error)
	HandleAttestations(context.Context, proto.Message) error
	IncomingAttFeed() *event.Feed
}

type powChainService interface {
	HasChainStartLogOccurred() (bool, uint64, error)
	ChainStartFeed() *event.Feed
	LatestBlockHeight() *big.Int
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
	BlockHashByHeight(ctx context.Context, height *big.Int) (common.Hash, error)
	BlockTimeByHeight(ctx context.Context, height *big.Int) (uint64, error)
	DepositRoot() [32]byte
	DepositTrie() *trieutil.MerkleTrie
	ChainStartDeposits() [][]byte
}

type syncService interface {
	Status() error
}

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	beaconDB            *db.BeaconDB
	chainService        chainService
	powChainService     powChainService
	operationService    operationService
	syncService         syncService
	port                string
	listener            net.Listener
	withCert            string
	withKey             string
	grpcServer          *grpc.Server
	canonicalStateChan  chan *pbp2p.BeaconState
	incomingAttestation chan *pbp2p.Attestation
	credentialError     error
	p2p                 p2p.Broadcaster
}

// Config options for the beacon node RPC server.
type Config struct {
	Port             string
	CertFlag         string
	KeyFlag          string
	BeaconDB         *db.BeaconDB
	ChainService     chainService
	POWChainService  powChainService
	OperationService operationService
	SyncService      syncService
	Broadcaster      p2p.Broadcaster
}

// NewRPCService creates a new instance of a struct implementing the BeaconServiceServer
// interface.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                 ctx,
		cancel:              cancel,
		beaconDB:            cfg.BeaconDB,
		p2p:                 cfg.Broadcaster,
		chainService:        cfg.ChainService,
		powChainService:     cfg.POWChainService,
		operationService:    cfg.OperationService,
		syncService:         cfg.SyncService,
		port:                cfg.Port,
		withCert:            cfg.CertFlag,
		withKey:             cfg.KeyFlag,
		canonicalStateChan:  make(chan *pbp2p.BeaconState, params.BeaconConfig().DefaultBufferSize),
		incomingAttestation: make(chan *pbp2p.Attestation, params.BeaconConfig().DefaultBufferSize),
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
		powChainService:     s.powChainService,
		chainService:        s.chainService,
		operationService:    s.operationService,
		incomingAttestation: s.incomingAttestation,
		canonicalStateChan:  s.canonicalStateChan,
		chainStartChan:      make(chan time.Time, 1),
	}
	proposerServer := &ProposerServer{
		beaconDB:           s.beaconDB,
		chainService:       s.chainService,
		powChainService:    s.powChainService,
		operationService:   s.operationService,
		canonicalStateChan: s.canonicalStateChan,
	}
	attesterServer := &AttesterServer{
		beaconDB:         s.beaconDB,
		operationService: s.operationService,
		p2p:              s.p2p,
	}
	validatorServer := &ValidatorServer{
		ctx:                s.ctx,
		beaconDB:           s.beaconDB,
		chainService:       s.chainService,
		canonicalStateChan: s.canonicalStateChan,
		powChainService:    s.powChainService,
	}
	pb.RegisterBeaconServiceServer(s.grpcServer, beaconServer)
	pb.RegisterProposerServiceServer(s.grpcServer, proposerServer)
	pb.RegisterAttesterServiceServer(s.grpcServer, attesterServer)
	pb.RegisterValidatorServiceServer(s.grpcServer, validatorServer)

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
