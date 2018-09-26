package beacon

import (
	"bytes"
	"context"
	"io"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/validator/params"
	"github.com/prysmaticlabs/prysm/validator/utils"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "beacon")

type rpcClientService interface {
	BeaconServiceClient() pb.BeaconServiceClient
}

// Service that interacts with a beacon node via RPC.
type Service struct {
	ctx                      context.Context
	cancel                   context.CancelFunc
	rpcClient                rpcClientService
	assignedSlot             uint64
	shardID                  uint64
	role                     pb.ValidatorRole
	attesterAssignmentFeed   *event.Feed
	proposerAssignmentFeed   *event.Feed
	processedAttestationFeed *event.Feed
	genesisTimestamp         time.Time
	slotAlignmentDuration    time.Duration
}

// NewBeaconValidator instantiates a service that interacts with a beacon node
// via gRPC requests.
func NewBeaconValidator(ctx context.Context, rpcClient rpcClientService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                      ctx,
		cancel:                   cancel,
		rpcClient:                rpcClient,
		attesterAssignmentFeed:   new(event.Feed),
		proposerAssignmentFeed:   new(event.Feed),
		processedAttestationFeed: new(event.Feed),
		slotAlignmentDuration:    time.Duration(params.DefaultConfig().SlotDuration) * time.Second,
	}
}

// Start the main routine for a beacon client service.
func (s *Service) Start() {
	log.Info("Starting service")
	beaconServiceClient := s.rpcClient.BeaconServiceClient()

	// First thing the validator does is request the current validator assignments
	// for the current beacon node cycle as well as the genesis timestamp
	// from the beacon node. From here, a validator can keep an internal
	// ticker that starts at the current slot the beacon node is in. This current slot
	// value is determined by taking the time differential between the genesis block
	// time and the current system time.
	//
	// Note: this does not validate the current system time against a global
	// NTP server, which will be important to do in production.
	// currently in a cycle we are supposed to participate in.
	s.fetchCurrentAssignmentsAndGenesisTime(beaconServiceClient)

	// Then, we kick off a routine that uses the begins a ticker based on the beacon node's
	// genesis timestamp and the validator will use this slot ticker to
	// determine when it is assigned to perform proposals or attestations.
	//
	// We block until the current time is a multiple of params.SlotDuration
	// so the validator and beacon node's internal tickers are aligned.
	utils.BlockingWait(s.slotAlignmentDuration)

	slotTicker := time.NewTicker(s.slotAlignmentDuration)
	go s.waitForAssignment(slotTicker.C, beaconServiceClient)

	// We then kick off a routine that listens for streams of cycle transitions
	// coming from the beacon node. This will allow the validator client to recalculate
	// when it has to perform its responsibilities.
	go s.listenForCycleTransitions(beaconServiceClient)
	go s.listenForProcessedAttestations(beaconServiceClient)
}

// Stop the main loop..
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// fetchCurrentAssignmentsAndGenesisTime fetches both the genesis timestamp as well
// as the current assignments for the current cycle in the beacon node. This allows
// the validator to do the following:
//
// (1) determine if it should act as an attester/proposer, at what slot,
// and what shard
//
// (2) determine the seconds since genesis by using the latest crystallized
// state recalc, then determine how many seconds have passed between that time
// and the current system time.
//
// From this, the validator client can deduce what slot interval the beacon
// node is in and determine when exactly it is time to propose or attest.
func (s *Service) fetchCurrentAssignmentsAndGenesisTime(client pb.BeaconServiceClient) {

	// Currently fetches assignments for all validators.
	req := &pb.ValidatorAssignmentRequest{
		AllValidators: true,
	}
	res, err := client.CurrentAssignmentsAndGenesisTime(s.ctx, req)
	if err != nil {
		// If this RPC request fails, the entire system should fatal as it is critical for
		// the validator to begin this way.
		log.Fatalf("could not fetch genesis time and latest canonical state from beacon node: %v", err)
	}

	// Determine what slot the beacon node is in by checking the number of seconds
	// since the genesis block.
	genesisTimestamp, err := ptypes.Timestamp(res.GetGenesisTimestamp())
	if err != nil {
		log.Fatalf("cannot compute genesis timestamp: %v", err)
	}

	s.genesisTimestamp = genesisTimestamp

	// Loops through the received assignments to determine which one
	// corresponds to this validator client based on a matching public key.
	for _, assignment := range res.GetAssignments() {
		// TODO(#566): Determine assignment based on public key flag.
		pubKeyProto := assignment.GetPublicKey()
		if isZeroAddress(pubKeyProto.GetPublicKey()) {
			s.assignedSlot = s.CurrentCycleStartSlot() + assignment.GetAssignedSlot()
			s.shardID = assignment.GetShardId()
			s.role = assignment.GetRole()
			break
		}
	}
	if s.role == pb.ValidatorRole_PROPOSER {
		log.Infof("Assigned as PROPOSER to slot %v, shardID: %v", s.assignedSlot, s.shardID)
	} else {
		log.Infof("Assigned as ATTESTER to slot %v, shardID: %v", s.assignedSlot, s.shardID)
	}
}

// waitForAssignment kicks off once the validator determines the currentSlot of the
// beacon node by calculating the difference between the current system time
// and the genesis timestamp. It runs exactly every SLOT_LENGTH seconds
// and checks if it is time for the validator to act as a proposer or attester.
func (s *Service) waitForAssignment(ticker <-chan time.Time, client pb.BeaconServiceClient) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker:
			log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("New beacon node slot")
			if s.role == pb.ValidatorRole_PROPOSER && s.assignedSlot == s.CurrentBeaconSlot() {
				log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("Assigned proposal slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the proposer service via a feed.
				s.proposerAssignmentFeed.Send(block)
			} else if s.role == pb.ValidatorRole_ATTESTER && s.assignedSlot == s.CurrentBeaconSlot() {
				log.Info("Assigned attestation slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the attester service a feed.
				s.attesterAssignmentFeed.Send(block)
			}
		}
	}
}

// listenForCycleTransitions receives validator assignments from the
// the beacon node's RPC server when a new cycle transition occurs.
func (s *Service) listenForCycleTransitions(client pb.BeaconServiceClient) {
	// Currently fetches assignments for all validators.
	req := &pb.ValidatorAssignmentRequest{
		AllValidators: true,
	}
	stream, err := client.ValidatorAssignments(s.ctx, req)
	if err != nil {
		log.Errorf("Could not setup validator assignments streaming client: %v", err)
		return
	}
	for {
		res, err := stream.Recv()

		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if s.ctx.Err() != nil {
			log.Debugf("Context has been canceled so shutting down the loop: %v", s.ctx.Err())
			break
		}

		if err != nil {
			log.Errorf("Could not receive validator assignments from stream: %v", err)
			continue
		}

		log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("New cycle transition")

		// Loops through the received assignments to determine which one
		// corresponds to this validator client based on a matching public key.
		for _, assignment := range res.GetAssignments() {
			// TODO(#566): Determine assignment based on public key flag.
			pubKeyProto := assignment.GetPublicKey()
			if isZeroAddress(pubKeyProto.GetPublicKey()) {
				s.assignedSlot = s.CurrentCycleStartSlot() + assignment.GetAssignedSlot()
				s.shardID = assignment.GetShardId()
				s.role = assignment.GetRole()
				break
			}
		}
		if s.role == pb.ValidatorRole_PROPOSER {
			log.Infof("Assigned as PROPOSER to slot %v, shardID: %v", s.assignedSlot, s.shardID)
		} else {
			log.Infof("Assigned as ATTESTER to slot %v, shardID: %v", s.assignedSlot, s.shardID)
		}
	}
}

// listenForProcessedAttestations receives processed attestations from the
// the beacon node's RPC server via gRPC streams.
func (s *Service) listenForProcessedAttestations(client pb.BeaconServiceClient) {
	stream, err := client.LatestAttestation(s.ctx, &empty.Empty{})
	if err != nil {
		log.Errorf("Could not setup beacon chain attestation streaming client: %v", err)
		return
	}
	for {
		attestation, err := stream.Recv()

		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		// If context is canceled we stop the loop.
		if s.ctx.Err() != nil {
			log.Debugf("Context has been canceled so shutting down the loop: %v", s.ctx.Err())
			break
		}
		if err != nil {
			log.Errorf("Could not receive latest attestation from stream: %v", err)
			continue
		}
		log.WithField("slotNumber", attestation.GetSlot()).Info("Latest attestation slot number")
		s.processedAttestationFeed.Send(attestation)
	}
}

// AttesterAssignmentFeed returns a feed that is written to whenever it is the validator's
// slot to perform attestations.
func (s *Service) AttesterAssignmentFeed() *event.Feed {
	return s.attesterAssignmentFeed
}

// ProposerAssignmentFeed returns a feed that is written to whenever it is the validator's
// slot to proposer blocks.
func (s *Service) ProposerAssignmentFeed() *event.Feed {
	return s.proposerAssignmentFeed
}

// ProcessedAttestationFeed returns a feed that is written to whenever an attestation
// is processed by a beacon node.
func (s *Service) ProcessedAttestationFeed() *event.Feed {
	return s.processedAttestationFeed
}

// CurrentBeaconSlot based on the genesis timestamp of the protocol.
func (s *Service) CurrentBeaconSlot() uint64 {
	secondsSinceGenesis := time.Since(s.genesisTimestamp).Seconds()
	return uint64(math.Floor(secondsSinceGenesis / params.DefaultConfig().SlotDuration))
}

// CurrentCycleStartSlot returns the slot at which the current cycle started.
func (s *Service) CurrentCycleStartSlot() uint64 {
	currentSlot := s.CurrentBeaconSlot()
	cycleNum := math.Floor(float64(currentSlot) / float64(params.DefaultConfig().CycleLength))
	return uint64(cycleNum) * params.DefaultConfig().CycleLength
}

// isZeroAddress compares a withdrawal address to an empty byte array.
func isZeroAddress(withdrawalAddress []byte) bool {
	return bytes.Equal(withdrawalAddress, []byte{})
}
