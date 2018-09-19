// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/beacon-chain/casper"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var log = logrus.WithField("prefix", "rpc")

type chainService interface {
	IncomingBlockFeed() *event.Feed
	IncomingAttestationFeed() *event.Feed
	CurrentCrystallizedState() *types.CrystallizedState
	ProcessedAttestationFeed() *event.Feed
}

// Service defining an RPC server for a beacon node.
type Service struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	announcer             types.CanonicalEventAnnouncer
	chainService          chainService
	port                  string
	listener              net.Listener
	withCert              string
	withKey               string
	grpcServer            *grpc.Server
	canonicalBlockChan    chan *types.Block
	canonicalStateChan    chan *types.CrystallizedState
	proccessedAttestation chan *pbp2p.AggregatedAttestation
}

// Config options for the beacon node RPC server.
type Config struct {
	Port            string
	CertFlag        string
	KeyFlag         string
	SubscriptionBuf int
	Announcer       types.CanonicalEventAnnouncer
	ChainService    chainService
}

// NewRPCService creates a new instance of a struct implementing the BeaconServiceServer
// interface.
func NewRPCService(ctx context.Context, cfg *Config) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                   ctx,
		cancel:                cancel,
		announcer:             cfg.Announcer,
		chainService:          cfg.ChainService,
		port:                  cfg.Port,
		withCert:              cfg.CertFlag,
		withKey:               cfg.KeyFlag,
		canonicalBlockChan:    make(chan *types.Block, cfg.SubscriptionBuf),
		canonicalStateChan:    make(chan *types.CrystallizedState, cfg.SubscriptionBuf),
		proccessedAttestation: make(chan *pbp2p.AggregatedAttestation, cfg.SubscriptionBuf),
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

	if s.withCert != "" && s.withKey != "" {
		creds, err := credentials.NewServerTLSFromFile(s.withCert, s.withKey)
		if err != nil {
			log.Errorf("could not load TLS keys: %s", err)
		}
		s.grpcServer = grpc.NewServer(grpc.Creds(creds))
	} else {
		log.Warn("You are using an insecure gRPC connection! Please provide a certificate and key to use a secure connection")
		s.grpcServer = grpc.NewServer()
	}

	pb.RegisterBeaconServiceServer(s.grpcServer, s)
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

// FetchShuffledValidatorIndices retrieves the shuffled validator indices, cutoffs, and
// assigned attestation slots at a given crystallized state hash.
// This function can be called by validators to fetch a historical list of shuffled
// validators ata point in time corresponding to a certain crystallized state.
func (s *Service) FetchShuffledValidatorIndices(ctx context.Context, req *pb.ShuffleRequest) (*pb.ShuffleResponse, error) {
	var shuffledIndices []uint64
	// Simulator always pushes out a validator list of length 100. By having index 0
	// as the last index, the validator will always be a proposer in the validator code.
	// TODO: Implement the real method by fetching the crystallized state in the request
	// from persistent disk storage and shuffling the indices appropriately.
	for i := 99; i >= 0; i-- {
		shuffledIndices = append(shuffledIndices, uint64(i))
	}
	// For now, this will cause validators to always pick the validator as a proposer.
	shuffleRes := &pb.ShuffleResponse{
		ShuffledValidatorIndices: shuffledIndices,
	}
	return shuffleRes, nil
}

// ProposeBlock is called by a proposer in a sharding validator and a full beacon node
// sends the request into a beacon block that can then be included in a canonical chain.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	// TODO: handle fields such as attestation bitmask, aggregate sig, and randao reveal.
	data := &pbp2p.BeaconBlock{
		SlotNumber: req.GetSlotNumber(),
		ParentHash: req.GetParentHash(),
		Timestamp:  req.GetTimestamp(),
	}
	block := types.NewBlock(data)
	h, err := block.Hash()
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}
	// We relay the received block from the proposer to the chain service for processing.
	s.chainService.IncomingBlockFeed().Send(block)
	return &pb.ProposeResponse{BlockHash: h[:]}, nil
}

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block.
func (s *Service) AttestHead(ctx context.Context, req *pb.AttestRequest) (*pb.AttestResponse, error) {
	attestation := types.NewAttestation(req.Attestation)
	h, err := attestation.Hash()
	if err != nil {
		return nil, fmt.Errorf("could not hash attestation: %v", err)
	}

	// Relays the attestation to chain service.
	s.chainService.IncomingAttestationFeed().Send(attestation)

	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}

// LatestBeaconBlock streams the latest beacon chain data.
func (s *Service) LatestBeaconBlock(req *empty.Empty, stream pb.BeaconService_LatestBeaconBlockServer) error {
	// Right now, this streams every announced block received via p2p. It should only stream
	// finalized blocks that are canonical in the beacon node after applying the fork choice
	// rule.
	sub := s.announcer.CanonicalBlockFeed().Subscribe(s.canonicalBlockChan)
	defer sub.Unsubscribe()
	for {
		select {
		case block := <-s.canonicalBlockChan:
			log.Info("Sending latest canonical block to RPC clients")
			if err := stream.Send(block.Proto()); err != nil {
				return err
			}
		case <-s.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// LatestCrystallizedState streams the latest beacon crystallized state.
func (s *Service) LatestCrystallizedState(req *empty.Empty, stream pb.BeaconService_LatestCrystallizedStateServer) error {
	// Right now, this streams every newly created crystallized state but should only
	// stream canonical states.
	sub := s.announcer.CanonicalCrystallizedStateFeed().Subscribe(s.canonicalStateChan)
	defer sub.Unsubscribe()
	for {
		select {
		case state := <-s.canonicalStateChan:
			log.Info("Sending crystallized state to RPC clients")
			if err := stream.Send(state.Proto()); err != nil {
				return err
			}
		case <-s.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// ValidatorShardID is called by a validator to get the shard ID of where it's suppose
// to proposer or attest.
func (s *Service) ValidatorShardID(ctx context.Context, req *pb.PublicKey) (*pb.ShardIDResponse, error) {
	cState := s.chainService.CurrentCrystallizedState()

	shardID, err := casper.ValidatorShardID(
		req.PublicKey,
		cState.CurrentDynasty(),
		cState.Validators(),
		cState.ShardAndCommitteesForSlots())
	if err != nil {
		return nil, fmt.Errorf("could not get validator shard ID: %v", err)
	}

	return &pb.ShardIDResponse{ShardId: shardID}, nil
}

// ValidatorSlot is called by a validator to get the slot number of when it's suppose
// to proposer or attest.
func (s *Service) ValidatorSlot(ctx context.Context, req *pb.PublicKey) (*pb.SlotResponse, error) {
	cState := s.chainService.CurrentCrystallizedState()

	slot, err := casper.ValidatorSlot(
		req.PublicKey,
		cState.CurrentDynasty(),
		cState.Validators(),
		cState.ShardAndCommitteesForSlots())
	if err != nil {
		return nil, fmt.Errorf("could not get validator slot for attester/propose: %v", err)
	}

	return &pb.SlotResponse{Slot: slot}, nil
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (s *Service) ValidatorIndex(ctx context.Context, req *pb.PublicKey) (*pb.IndexResponse, error) {
	cState := s.chainService.CurrentCrystallizedState()

	index, err := casper.ValidatorIndex(
		req.PublicKey,
		cState.CurrentDynasty(),
		cState.Validators())
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}

	return &pb.IndexResponse{Index: index}, nil
}

// LatestAttestation streams the latest processed attestations to the rpc clients.
func (s *Service) LatestAttestation(req *empty.Empty, stream pb.BeaconService_LatestAttestationServer) error {
	sub := s.chainService.ProcessedAttestationFeed().Subscribe(s.proccessedAttestation)
	defer sub.Unsubscribe()

	for {
		select {
		case attestation := <-s.proccessedAttestation:
			log.Info("Sending attestation to RPC clients")
			if err := stream.Send(attestation); err != nil {
				return err
			}
		case <-s.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}
