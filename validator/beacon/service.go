package beacon

import (
	"context"
	"github.com/prysmaticlabs/prysm/validator/params"
	"io"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
<<<<<<< HEAD
=======
	"github.com/prysmaticlabs/prysm/validator/params"
	"github.com/prysmaticlabs/prysm/validator/utils"
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "beacon")

type rpcClientService interface {
	BeaconServiceClient() pb.BeaconServiceClient
}

type validatorClientService interface {
	ValidatorServiceClient() pb.ValidatorServiceClient
}

// Service that interacts with a beacon node via RPC.
type Service struct {
	ctx                      context.Context
	cancel                   context.CancelFunc
	pubKey                   []byte
	rpcClient                rpcClientService
<<<<<<< HEAD
	validatorClient          validatorClientService
	assignedSlot             uint64
	assignedShardID          uint64
	responsibility           pb.ValidatorRole
=======
	assignedSlot             uint64
	shardID                  uint64
	role                     pb.ValidatorRole
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
	attesterAssignmentFeed   *event.Feed
	proposerAssignmentFeed   *event.Feed
	processedAttestationFeed *event.Feed
	genesisTimestamp         time.Time
	slotAlignmentDuration    time.Duration
}

// NewBeaconValidator instantiates a service that interacts with a beacon node
// via gRPC requests.
func NewBeaconValidator(ctx context.Context, pubKey []byte, rpcClient rpcClientService, validatorClient validatorClientService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                      ctx,
		pubKey:                   pubKey,
		cancel:                   cancel,
		rpcClient:                rpcClient,
		validatorClient:          validatorClient,
		attesterAssignmentFeed:   new(event.Feed),
		proposerAssignmentFeed:   new(event.Feed),
		processedAttestationFeed: new(event.Feed),
		slotAlignmentDuration:    time.Duration(params.DefaultConfig().SlotDuration) * time.Second,
	}
}

// Start the main routine for a beacon client service.
func (s *Service) Start() {
	log.Info("Starting service")
<<<<<<< HEAD
	client := s.rpcClient.BeaconServiceClient()
	validator := s.validatorClient.ValidatorServiceClient()
=======
	beaconServiceClient := s.rpcClient.BeaconServiceClient()
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a

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

<<<<<<< HEAD
	// We kick off a routine that listens for stream of validator assignment coming from
	// beacon node. This will update validator client on which slot, shard ID and what
	// responsbility to perform.
	go s.listenForAssignmentChange(validator)

	// Then, we kick off a routine that uses the begins a ticker set in fetchGenesisAndCanonicalState
	// to wait until the validator's assigned slot to perform proposals or attestations.
	ticker := time.NewTicker(time.Second * time.Duration(params.DefaultConfig().SlotDuration))
	go s.waitForAssignment(ticker.C, client)

	// Finally, we kick off a routine to pass processed attestations from beacon node to attester.
	go s.listenForProcessedAttestations(client)
=======
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
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
}

// Stop the main loop.
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

<<<<<<< HEAD
// CurrentBeaconSlot based on the seconds since genesis.
func (s *Service) CurrentBeaconSlot() uint64 {
	secondsSinceGenesis := time.Since(s.genesisTimestamp).Seconds()
	return uint64(math.Floor(secondsSinceGenesis / 8.0))
}

// fetchGenesisAndCanonicalState fetches the genesis timestamp.
func (s *Service) fetchGenesisAndCanonicalState(client pb.BeaconServiceClient) {
	res, err := client.GenesisStartTime(s.ctx, &empty.Empty{})
=======
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
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
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
<<<<<<< HEAD
}

// listenForAssignmentChange listens for validator assignment changes via a RPC stream.
// when there's an assignment change, beacon service will update its shard ID, slot number and role.
func (s *Service) listenForAssignmentChange(validator pb.ValidatorServiceClient) {
	req := &pb.ValidatorAssignmentRequest{PublicKeys: []*pb.PublicKey{{PublicKey: s.pubKey}}}
	stream, err := validator.ValidatorAssignment(s.ctx, req)
	if err != nil {
		log.Errorf("could not fetch validator assigned slot and responsibility from beacon node: %v", err)
=======

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
	if s.CurrentBeaconSlot() > s.assignedSlot {
		log.Info("You joined a bit too late -the current slot is greater than assigned slot in the cycle, wait until next cycle to be re-assigned")
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
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
		return
	}

	for {
<<<<<<< HEAD
		assignment, err := stream.Recv()
=======
		res, err := stream.Recv()

>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
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
<<<<<<< HEAD
			log.Errorf("Could not receive latest validator assignment from stream: %v", err)
			continue
		}
		s.responsibility = assignment.Assignments[0].Role
		s.assignedSlot = assignment.Assignments[0].AssignedSlot
		s.assignedShardID = assignment.Assignments[0].ShardId

		log.Infof("Validator with pub key 0x%s re-assigned to shard ID %d for %v duty at slot %d",
			string(assignment.Assignments[0].PublicKey.PublicKey),
			s.assignedShardID,
			s.responsibility,
			s.assignedSlot)
	}
}

// waitForAssignment waits till it's validator's role to attest or propose. Then it forwards
// the canonical block to the proposer service or attester service to process.
func (s *Service) waitForAssignment(ticker <-chan time.Time, client pb.BeaconServiceClient) {
	for {
		select {
		case <-s.ctx.Done():
			return

		case <-ticker:
			if s.responsibility == pb.ValidatorRole_ATTESTER && s.assignedSlot == s.CurrentBeaconSlot() {
				log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("Assigned attest slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the attester service a feed.
				s.attesterAssignmentFeed.Send(block)

			} else if s.responsibility == pb.ValidatorRole_PROPOSER && s.assignedSlot == s.CurrentBeaconSlot() {
				log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("Assigned proposal slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the proposer service a feed.
				s.proposerAssignmentFeed.Send(block)
			}
=======
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
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
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

<<<<<<< HEAD
// PublicKey returns validator's public key.
func (s *Service) PublicKey() []byte {
	return s.pubKey
=======
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
>>>>>>> 48c07bfeb9a2c86181cc4dab8404039031207b9a
}
