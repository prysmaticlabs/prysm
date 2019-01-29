// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var log = logrus.WithField("prefix", "rpc")

type chainService interface {
	IncomingBlockFeed() *event.Feed
	// These methods are not called on-demand by a validator
	// but instead streamed to connected validators every
	// time the canonical head changes in the chain service.
	CanonicalBlockFeed() *event.Feed
	CanonicalStateFeed() *event.Feed
}

type attestationService interface {
	IncomingAttestationFeed() *event.Feed
}

type powChainService interface {
	LatestBlockHash() common.Hash
}

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	beaconDB              *db.BeaconDB
	chainService          chainService
	powChainService       powChainService
	attestationService    attestationService
	port                  string
	listener              net.Listener
	withCert              string
	withKey               string
	grpcServer            *grpc.Server
	canonicalBlockChan    chan *pbp2p.BeaconBlock
	canonicalStateChan    chan *pbp2p.BeaconState
	incomingAttestation   chan *pbp2p.Attestation
	enablePOWChain        bool
	slotAlignmentDuration time.Duration
}

// Config options for the beacon node RPC server.
type Config struct {
	Port               string
	CertFlag           string
	KeyFlag            string
	SubscriptionBuf    int
	BeaconDB           *db.BeaconDB
	ChainService       chainService
	POWChainService    powChainService
	AttestationService attestationService
	EnablePOWChain     bool
}

// NewRPCService creates a new instance of a struct implementing the BeaconServiceServer
// interface.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		beaconDB:              cfg.BeaconDB,
		chainService:          cfg.ChainService,
		powChainService:       cfg.POWChainService,
		attestationService:    cfg.AttestationService,
		port:                  cfg.Port,
		withCert:              cfg.CertFlag,
		withKey:               cfg.KeyFlag,
		slotAlignmentDuration: time.Duration(params.BeaconConfig().SlotDuration) * time.Second,
		canonicalBlockChan:    make(chan *pbp2p.BeaconBlock, cfg.SubscriptionBuf),
		canonicalStateChan:    make(chan *pbp2p.BeaconState, cfg.SubscriptionBuf),
		incomingAttestation:   make(chan *pbp2p.Attestation, cfg.SubscriptionBuf),
		enablePOWChain:        cfg.EnablePOWChain,
	}
}

// Start the gRPC server.
func (s *Service) Start() {
	log.Info("Starting service")

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		log.Errorf("Could not listen to port :%s: %v", s.port, err)
		return
	}
	s.listener = lis
	log.Infof("RPC server listening on port :%s", s.port)

	// TODO(#791): Utilize a certificate for secure connections
	// between beacon nodes and validator clients.
	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("Could not load TLS keys: %s", err)
		}
		s.grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC connection! Provide a certificate and key to connect securely")
		s.grpcServer = grpc.NewServer()
	}

	beaconServer := &BeaconServer{
		beaconDB:            s.beaconDB,
		ctx:                 s.ctx,
		attestationService:  s.attestationService,
		incomingAttestation: s.incomingAttestation,
		canonicalStateChan:  s.canonicalStateChan,
	}
	proposerServer := &ProposerServer{
		beaconDB:           s.beaconDB,
		chainService:       s.chainService,
		powChainService:    s.powChainService,
		canonicalStateChan: s.canonicalStateChan,
		enablePOWChain:     s.enablePOWChain,
	}
	attesterServer := &AttesterServer{
		attestationService: s.attestationService,
	}
	validatorServer := &ValidatorServer{
		beaconDB: s.beaconDB,
	}
	pb.RegisterBeaconServiceServer(s.grpcServer, beaconServer)
	pb.RegisterProposerServiceServer(s.grpcServer, proposerServer)
	pb.RegisterAttesterServiceServer(s.grpcServer, attesterServer)
	pb.RegisterValidatorServiceServer(s.grpcServer, validatorServer)

	// Register reflection service on gRPC server.
	reflection.Register(s.grpcServer)

	go func() {
		err = s.grpcServer.Serve(lis)
		if err != nil {
			log.Errorf("Could not serve gRPC: %v", err)
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

// Status always returns nil.
// TODO(1205): Add service health checks.
func (s *Service) Status() error {
	return nil
}
