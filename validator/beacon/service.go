package beacon

import (
	"bytes"
	"context"
	"io"
	"math"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
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
	pubKey                   []byte
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
func NewBeaconValidator(ctx context.Context, pubKey []byte, rpcClient rpcClientService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                      ctx,
		pubKey:                   pubKey,
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

	// We kick off a routine that listens for stream of validator assignment coming from
	// beacon node. This will update validator client on which slot, shard ID and what
	// responsbility to perform.
	go s.listenForAssignmentChange(beaconServiceClient)

	slotTicker := time.NewTicker(s.slotAlignmentDuration)
	go s.waitForAssignment(slotTicker.C, beaconServiceClient)

	go s.listenForProcessedAttestations(beaconServiceClient)
}

// Stop the main loop.
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

	log.Infof("Setting validator genesis time to %s", genesisTimestamp.Format(time.UnixDate))
	s.genesisTimestamp = genesisTimestamp
	for _, assign := range res.Assignments {
		if bytes.Equal(assign.PublicKey.PublicKey, s.pubKey) {
			s.role = assign.Role
			// + 1 to account for the genesis block being slot 0.
			s.assignedSlot = s.CurrentCycleStartSlot(params.DemoConfig().CycleLength) + assign.AssignedSlot + 1
			s.shardID = assign.ShardId

			log.Infof("Validator shuffled. Pub key 0x%s re-assigned to shard ID %d for %v duty at slot %d",
				string(s.pubKey),
				s.shardID,
				s.role,
				s.assignedSlot)
		}
	}
}

// listenForAssignmentChange listens for validator assignment changes via a RPC stream.
// when there's an assignment change, beacon service will update its shard ID, slot number and role.
func (s *Service) listenForAssignmentChange(client pb.BeaconServiceClient) {
	req := &pb.ValidatorAssignmentRequest{PublicKeys: []*pb.PublicKey{{PublicKey: s.pubKey}}}
	stream, err := client.ValidatorAssignments(s.ctx, req)
	if err != nil {
		log.Errorf("could not fetch validator assigned slot and responsibility from beacon node: %v", err)
		return
	}
	for {
		assignment, err := stream.Recv()
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
			log.Errorf("Could not receive latest validator assignment from stream: %v", err)
			continue
		}

		for _, assign := range assignment.Assignments {
			if bytes.Equal(assign.PublicKey.PublicKey, s.pubKey) {
				s.role = assign.Role
				if s.CurrentCycleStartSlot(params.DemoConfig().CycleLength) == 0 {
					// +1 to account for genesis block being slot 0.
					s.assignedSlot = params.DemoConfig().CycleLength + assign.AssignedSlot + 1
				} else {
					s.assignedSlot = s.CurrentCycleStartSlot(params.DemoConfig().CycleLength) + assign.AssignedSlot + 1
				}
				s.shardID = assign.ShardId

				log.Infof("Validator with pub key 0x%s re-assigned to shard ID %d for %v duty at slot %d",
					string(s.pubKey),
					s.shardID,
					s.role,
					s.assignedSlot)
			}
		}
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
			currentSlot := s.CurrentBeaconSlot()
			log.Infof("role: %v, assigned slot: %d, current slot: %d", s.role, s.assignedSlot, currentSlot)
			if s.role == pb.ValidatorRole_ATTESTER && s.assignedSlot == currentSlot {
				log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("Assigned attest slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the attester service a feed.
				s.attesterAssignmentFeed.Send(block)

			} else if s.role == pb.ValidatorRole_PROPOSER && s.assignedSlot == currentSlot {
				log.WithField("slotNumber", s.CurrentBeaconSlot()).Info("Assigned proposal slot number reached")
				block, err := client.CanonicalHead(s.ctx, &empty.Empty{})
				if err != nil {
					log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
					continue
				}
				// We forward the latest canonical block to the proposer service a feed.
				s.proposerAssignmentFeed.Send(block)
			}
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

// PublicKey returns validator's public key.
func (s *Service) PublicKey() []byte {
	return s.pubKey
}

// CurrentBeaconSlot based on the genesis timestamp of the protocol.
func (s *Service) CurrentBeaconSlot() uint64 {
	secondsSinceGenesis := time.Since(s.genesisTimestamp).Seconds()
	if secondsSinceGenesis-params.DefaultConfig().SlotDuration < 0 {
		return 0
	}
	return uint64(math.Floor(secondsSinceGenesis/params.DefaultConfig().SlotDuration)) - 1
}

// CurrentCycleStartSlot returns the slot at which the current cycle started.
func (s *Service) CurrentCycleStartSlot(cycleLength uint64) uint64 {
	currentSlot := s.CurrentBeaconSlot()
	cycleNum := currentSlot / cycleLength
	return uint64(cycleNum) * cycleLength
}
