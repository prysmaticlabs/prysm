// Package rpc defines the services that the beacon-chain uses to communicate via gRPC.
package rpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	pb.RegisterBeaconServiceServer(s.grpcServer, s)
	pb.RegisterValidatorServiceServer(s.grpcServer, s)
	pb.RegisterProposerServiceServer(s.grpcServer, s)
	pb.RegisterAttesterServiceServer(s.grpcServer, s)
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

// CanonicalHead of the current beacon chain. This method is requested on-demand
// by a validator when it is their time to propose or attest.
func (s *Service) CanonicalHead(ctx context.Context, req *ptypes.Empty) (*pbp2p.BeaconBlock, error) {
	block, err := s.beaconDB.GetChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get canonical head block: %v", err)
	}
	return block, nil
}

// CurrentAssignmentsAndGenesisTime returns the current validator assignments
// based on the beacon node's current, canonical crystallized state.
// Validator clients send this request once upon establishing a connection
// to the beacon node in order to determine their role and assigned slot
// initially. This method also returns the genesis timestamp
// of the beacon node which will allow a validator client to setup a
// a ticker to keep track of the current beacon slot.
func (s *Service) CurrentAssignmentsAndGenesisTime(
	ctx context.Context,
	req *pb.ValidatorAssignmentRequest,
) (*pb.CurrentAssignmentsResponse, error) {
	genesis, err := s.beaconDB.GetBlockBySlot(0)
	if err != nil {
		return nil, fmt.Errorf("could not get genesis block: %v", err)
	}
	beaconState, err := s.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	var keys []*pb.PublicKey
	if req.AllValidators {
		for _, val := range beaconState.GetValidatorRegistry() {
			keys = append(keys, &pb.PublicKey{PublicKey: val.GetPubkey()})
		}
	} else {
		keys = req.GetPublicKeys()
		if len(keys) == 0 {
			return nil, errors.New("no public keys specified in request")
		}
	}
	assignments, err := assignmentsForPublicKeys(keys, beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not get assignments for public keys: %v", err)
	}

	return &pb.CurrentAssignmentsResponse{
		GenesisTimestamp: genesis.GetTimestamp(),
		Assignments:      assignments,
	}, nil
}

// ProposeBlock is called by a proposer in a sharding validator and a full beacon node
// sends the request into a beacon block that can then be included in a canonical chain.
func (s *Service) ProposeBlock(ctx context.Context, req *pb.ProposeRequest) (*pb.ProposeResponse, error) {
	var powChainHash common.Hash
	if !s.enablePOWChain {
		powChainHash = common.BytesToHash([]byte{byte(req.GetSlotNumber())})
	} else {
		powChainHash = s.powChainService.LatestBlockHash()
	}

	//TODO(#589) The attestation should be aggregated in the validator client side not in the beacon node.
	beaconState, err := s.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	_, prevProposerIndex, err := v.ProposerShardAndIndex(
		beaconState.GetShardAndCommitteesAtSlots(),
		beaconState.GetLastStateRecalculationSlot(),
		req.GetSlotNumber(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not get index of previous proposer: %v", err)
	}

	proposerBitfield := uint64(math.Pow(2, (7 - float64(prevProposerIndex))))
	attestation := &pbp2p.Attestation{
		ParticipationBitfield: []byte{byte(proposerBitfield)},
	}

	block := &pbp2p.BeaconBlock{
		Slot:                          req.GetSlotNumber(),
		CandidatePowReceiptRootHash32: powChainHash[:],
		ParentRootHash32:              req.GetParentHash(),
		Timestamp:                     req.GetTimestamp(),
		Body: &pbp2p.BeaconBlockBody{
			Attestations: []*pbp2p.Attestation{attestation},
		},
	}

	h, err := b.Hash(block)
	if err != nil {
		return nil, fmt.Errorf("could not hash block: %v", err)
	}
	log.WithField("blockHash", fmt.Sprintf("%#x", h)).Debugf("Block proposal received via RPC")
	// We relay the received block from the proposer to the chain service for processing.
	s.chainService.IncomingBlockFeed().Send(block)
	return &pb.ProposeResponse{BlockHash: h[:]}, nil
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

// LatestAttestation streams the latest processed attestations to the rpc clients.
func (s *Service) LatestAttestation(req *ptypes.Empty, stream pb.BeaconService_LatestAttestationServer) error {
	sub := s.attestationService.IncomingAttestationFeed().Subscribe(s.incomingAttestation)
	defer sub.Unsubscribe()
	for {
		select {
		case attestation := <-s.incomingAttestation:
			log.Info("Sending attestation to RPC clients")
			if err := stream.Send(attestation); err != nil {
				return err
			}
		case <-sub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return nil
		case <-s.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// ValidatorShardID is called by a validator to get the shard ID of where it's suppose
// to proposer or attest.
func (s *Service) ValidatorShardID(ctx context.Context, req *pb.PublicKey) (*pb.ShardIDResponse, error) {
	beaconState, err := s.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	shardID, err := v.ValidatorShardID(
		req.PublicKey,
		beaconState.GetValidatorRegistry(),
		beaconState.GetShardAndCommitteesAtSlots(),
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
	beaconState, err := s.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}

	slot, role, err := v.ValidatorSlotAndRole(
		req.PublicKey,
		beaconState.GetValidatorRegistry(),
		beaconState.GetShardAndCommitteesAtSlots(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not get assigned validator slot for attester/proposer: %v", err)
	}

	return &pb.SlotResponsibilityResponse{Slot: slot, Role: role}, nil
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (s *Service) ValidatorIndex(ctx context.Context, req *pb.PublicKey) (*pb.IndexResponse, error) {
	beaconState, err := s.beaconDB.GetState()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	index, err := v.ValidatorIndex(
		req.PublicKey,
		beaconState.GetValidatorRegistry(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}

	return &pb.IndexResponse{Index: index}, nil
}

// ValidatorAssignments streams validator assignments every cycle transition
// to clients that request to watch a subset of public keys in the
// CrystallizedState's active validator set.
func (s *Service) ValidatorAssignments(
	req *pb.ValidatorAssignmentRequest,
	stream pb.BeaconService_ValidatorAssignmentsServer) error {

	sub := s.chainService.CanonicalStateFeed().Subscribe(s.canonicalStateChan)
	defer sub.Unsubscribe()
	for {
		select {
		case beaconState := <-s.canonicalStateChan:
			log.Info("Sending new cycle assignments to validator clients")

			var keys []*pb.PublicKey
			if req.AllValidators {
				for _, val := range beaconState.GetValidatorRegistry() {
					keys = append(keys, &pb.PublicKey{PublicKey: val.GetPubkey()})
				}
			} else {
				keys = req.GetPublicKeys()
				if len(keys) == 0 {
					return errors.New("no public keys specified in request")
				}
			}

			assignments, err := assignmentsForPublicKeys(keys, beaconState)
			if err != nil {
				return fmt.Errorf("could not get assignments for public keys: %v", err)
			}

			// We create a response consisting of all the assignments for each
			// corresponding, valid public key in the request. We also include
			// the beacon node's current beacon slot in the response.
			res := &pb.ValidatorAssignmentResponse{
				Assignments: assignments,
			}
			if err := stream.Send(res); err != nil {
				return err
			}
		case <-sub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return nil
		case <-s.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// assignmentsForPublicKeys fetches the validator assignments for a subset of public keys
// given a crystallized state.
func assignmentsForPublicKeys(keys []*pb.PublicKey, beaconState *pbp2p.BeaconState) ([]*pb.Assignment, error) {
	// Next, for each public key in the request, we build
	// up an array of assignments.
	assignments := []*pb.Assignment{}
	for _, val := range keys {
		// For the corresponding public key and current crystallized state,
		// we determine the assigned slot for the validator and whether it
		// should act as a proposer or attester.
		assignedSlot, role, err := v.ValidatorSlotAndRole(
			val.GetPublicKey(),
			beaconState.GetValidatorRegistry(),
			beaconState.GetShardAndCommitteesAtSlots(),
		)
		if err != nil {
			return nil, err
		}

		// We determine the assigned shard ID for the validator
		// based on a public key and current crystallized state.
		shardID, err := v.ValidatorShardID(
			val.GetPublicKey(),
			beaconState.GetValidatorRegistry(),
			beaconState.GetShardAndCommitteesAtSlots(),
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, &pb.Assignment{
			PublicKey:    val,
			ShardId:      shardID,
			Role:         role,
			AssignedSlot: assignedSlot,
		})
	}
	return assignments, nil
}
