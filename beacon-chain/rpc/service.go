// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytes"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
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
		beaconDB: s.beaconDB,
		ctx: s.ctx,
		attestationService: s.attestationService,
		incomingAttestation: s.incomingAttestation,
		canonicalStateChan: s.canonicalStateChan,
	}
	pb.RegisterBeaconServiceServer(s.grpcServer, beaconServer)
	pb.RegisterValidatorServiceServer(s.grpcServer, s)
	pb.RegisterProposerServiceServer(s.grpcServer, s)
	pb.RegisterAttesterServiceServer(s.grpcServer, s)

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

// ProposeBlock is called by a proposer in a sharding validator and a full beacon node
// sends the request into a beacon block that can then be included in a canonical chain.
func (s *Service) ProposeBlock(ctx context.Context, blk *pbp2p.BeaconBlock) (*pb.ProposeResponse, error) {

	h, err := hashutil.HashBeaconBlock(blk)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("%#x", h)).Debugf("Block proposal received via RPC")
	// We relay the received block from the proposer to the chain service for processing.
	s.chainService.IncomingBlockFeed().Send(blk)
	return &pb.ProposeResponse{BlockHash: h[:]}, nil
}

// LatestPOWChainBlockHash retrieves the latest blockhash of the POW chain and sends it to the validator client.
func (s *Service) LatestPOWChainBlockHash(ctx context.Context, req *ptypes.Empty) (*pb.POWChainResponse, error) {
	var powChainHash common.Hash

	if !s.enablePOWChain {
		powChainHash = common.BytesToHash([]byte{'p', 'o', 'w', 'c', 'h', 'a', 'i', 'n'})

		return &pb.POWChainResponse{
			BlockHash: powChainHash[:],
		}, nil
	}

	powChainHash = s.powChainService.LatestBlockHash()

	return &pb.POWChainResponse{
		BlockHash: powChainHash[:],
	}, nil

}

// ComputeStateRoot computes the state root after a block has been processed through a state transition and
// returns it to the validator client.
func (s *Service) ComputeStateRoot(ctx context.Context, req *pbp2p.BeaconBlock) (*pb.StateRootResponse, error) {

	beaconState, err := s.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	parentHash := bytes.ToBytes32(req.ParentRootHash32)

	beaconState, err = state.ExecuteStateTransition(beaconState, req, parentHash)
	if err != nil {
		return nil, fmt.Errorf("could not execute state transition %v", err)
	}

	encodedState, err := proto.Marshal(beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not marshal state %v", err)
	}

	beaconStateHash := hashutil.Hash(encodedState)

	log.WithField("beaconStateHash", fmt.Sprintf("%#x", beaconStateHash)).Debugf("Computed state hash")

	return &pb.StateRootResponse{
		StateRoot: beaconStateHash[:],
	}, nil
}

// ProposerIndex sends a response to the client which returns the proposer index for a given slot. Validators
// are shuffled and assigned slots to attest/propose to. This method will look for the validator that is assigned
// to propose a beacon block at the given slot.
func (s *Service) ProposerIndex(ctx context.Context, req *pb.ProposerIndexRequest) (*pb.IndexResponse, error) {

	beaconState, err := s.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	_, ProposerIndex, err := v.ProposerShardAndIdx(
		beaconState,
		req.SlotNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get index of previous proposer: %v", err)
	}

	return &pb.IndexResponse{
		Index: uint32(ProposerIndex),
	}, nil
}

// AttestHead is a function called by an attester in a sharding validator to vote
// on a block.
func (s *Service) AttestHead(ctx context.Context, req *pb.AttestRequest) (*pb.AttestResponse, error) {
	enc, err := proto.Marshal(req.Attestation)
	if err != nil {
		return nil, fmt.Errorf("could not marshal attestation: %v", err)
	}
	h := hashutil.Hash(enc)
	// Relays the attestation to chain service.
	s.attestationService.IncomingAttestationFeed().Send(req.Attestation)

	return &pb.AttestResponse{AttestationHash: h[:]}, nil
}

// ValidatorShardID is called by a validator to get the shard ID of where it's suppose
// to proposer or attest.
func (s *Service) ValidatorShardID(ctx context.Context, req *pb.PublicKey) (*pb.ShardIDResponse, error) {
	beaconState, err := s.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	shardID, err := v.ValidatorShardID(
		req.PublicKey,
		beaconState.ValidatorRegistry,
		beaconState.ShardCommitteesAtSlots,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get validator shard ID: %v", err)
	}

	return &pb.ShardIDResponse{ShardId: shardID}, nil
}

// ValidatorSlotAndResponsibility fetches a validator's assigned slot number
// and whether it should act as a proposer/attester.
func (s *Service) ValidatorSlotAndResponsibility(
	ctx context.Context,
	req *pb.PublicKey,
) (*pb.SlotResponsibilityResponse, error) {
	beaconState, err := s.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	slot, role, err := v.ValidatorSlotAndRole(
		req.PublicKey,
		beaconState.ValidatorRegistry,
		beaconState.ShardCommitteesAtSlots,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get assigned validator slot for attester/proposer: %v", err)
	}

	return &pb.SlotResponsibilityResponse{Slot: slot, Role: role}, nil
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (s *Service) ValidatorIndex(ctx context.Context, req *pb.PublicKey) (*pb.IndexResponse, error) {
	beaconState, err := s.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	index, err := v.ValidatorIdx(
		req.PublicKey,
		beaconState.ValidatorRegistry,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}

	return &pb.IndexResponse{Index: index}, nil
}

// ValidatorEpochAssignments ... WIP
func (s *Service) ValidatorEpochAssignments(ctx context.Context, req *pb.ValidatorEpochAssignmentsRequest) (*pb.ValidatorEpochAssignmentsResponse, error) {
	return nil, nil
}

