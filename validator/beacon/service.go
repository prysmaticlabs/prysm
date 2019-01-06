package beacon

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotticker"
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
	if err := s.fetchCurrentAssignmentsAndGenesisTime(beaconServiceClient); err != nil {
		log.Error(err)
		return
	}

	// We kick off a routine that listens for stream of validator assignment coming from
	// beacon node. This will update validator client on which slot, shard ID and what
	// responsbility to perform.
	go s.listenForAssignmentChange(beaconServiceClient)

	slotTicker := slotticker.GetSlotTicker(s.genesisTimestamp, params.BeaconConfig().SlotDuration)
	go func() {
		s.waitForAssignment(slotTicker.C(), beaconServiceClient)
		slotTicker.Done()
	}()

	go s.listenForProcessedAttestations(beaconServiceClient)
}

// Stop the main loop.
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// Status always returns nil.
// This service will be rewritten in the future so this service check is a
// no-op for now.
func (s *Service) Status() error {
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
func (s *Service) fetchCurrentAssignmentsAndGenesisTime(client pb.BeaconServiceClient) error {
	// Currently fetches assignments for all validators.
	req := &pb.ValidatorAssignmentRequest{
		AllValidators: true,
	}
	res, err := client.CurrentAssignmentsAndGenesisTime(s.ctx, req)
	if err != nil {
		// If this RPC request fails, the entire system should fatal as it is critical for
		// the validator to begin this way.
		return fmt.Errorf("could not fetch genesis time and latest canonical state from beacon node: %v", err)
	}

	// Determine what slot the beacon node is in by checking the number of seconds
	// since the genesis block.
	genesisTimestamp, err := ptypes.TimestampFromProto(res.GenesisTimestamp)
	if err != nil {
		return fmt.Errorf("could not compute genesis timestamp: %v", err)
	}
	s.genesisTimestamp = genesisTimestamp

	startSlot := s.startSlot()
	if err := s.assignRole(res.Assignments, startSlot); err != nil {
		return fmt.Errorf("unable to assign a role: %v", err)
	}
	return nil
}

// listenForAssignmentChange listens for validator assignment changes via a RPC stream.
// when there's an assignment change, beacon service will update its shard ID, slot number and role.
func (s *Service) listenForAssignmentChange(client pb.BeaconServiceClient) {
	req := &pb.ValidatorAssignmentRequest{PublicKeys: []*pb.PublicKey{{PublicKey: s.pubKey}}}
	stream, err := client.ValidatorAssignments(s.ctx, req)
	if err != nil {
		log.Errorf("failed to fetch validator assignments stream: %v", err)
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
			break
		}

		startSlot := s.startSlot()
		if err := s.assignRole(assignment.Assignments, startSlot); err != nil {
			log.Errorf("Could not assign a role for validator: %v", err)
			break
		}
	}
}

// waitForAssignment waits till it's validator's role to attest or propose. Then it forwards
// the canonical block to the proposer service or attester service to process.
func (s *Service) waitForAssignment(ticker <-chan uint64, client pb.BeaconServiceClient) {
	for {
		select {
		case <-s.ctx.Done():
			return

		case slot := <-ticker:
			log = log.WithField("slot", slot)
			log.Infof("tick")

			// Special case: skip responsibilities if assigned to the genesis block.
			if s.assignedSlot != slot || s.assignedSlot == 0 {
				continue
			}

			block, err := client.CanonicalHead(s.ctx, &ptypes.Empty{})
			if err != nil {
				log.Errorf("Could not fetch canonical head via gRPC from beacon node: %v", err)
				continue
			}

			if s.role == pb.ValidatorRole_ATTESTER {
				log.Info("Assigned attestation slot number reached")
				// We forward the latest canonical block to the attester service a feed.
				s.attesterAssignmentFeed.Send(block)
			} else if s.role == pb.ValidatorRole_PROPOSER {
				log.Info("Assigned proposal slot number reached")
				// We forward the latest canonical block to the proposer service a feed.
				s.proposerAssignmentFeed.Send(block)
			}
		}
	}
}

// listenForProcessedAttestations receives processed attestations from the
// the beacon node's RPC server via gRPC streams.
func (s *Service) listenForProcessedAttestations(client pb.BeaconServiceClient) {
	stream, err := client.LatestAttestation(s.ctx, &ptypes.Empty{})
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

		log.WithField("slotNumber", attestation.Data.Slot).Info("Latest attestation slot number")
		s.processedAttestationFeed.Send(attestation)
	}
}

// startSlot returns the first slot of the given slot's cycle.
func (s *Service) startSlot() uint64 {
	duration := params.BeaconConfig().SlotDuration
	epochLength := params.BeaconConfig().EpochLength
	slot := slotticker.CurrentSlot(s.genesisTimestamp, duration, time.Since)
	return slot - slot%epochLength
}

func (s *Service) assignRole(assignments []*pb.Assignment, startSlot uint64) error {
	var role pb.ValidatorRole
	var assignedSlot uint64
	var shardID uint64
	for _, assign := range assignments {
		if !bytes.Equal(assign.PublicKey.PublicKey, s.pubKey) {
			continue
		}

		role = assign.Role
		assignedSlot = startSlot + assign.AssignedSlot
		shardID = assign.ShardId

		log.Infof("Validator shuffled. Pub key %#x assigned to shard ID %d for %v duty at slot %d",
			s.pubKey,
			shardID,
			role,
			assignedSlot)

		break
	}

	if role == pb.ValidatorRole_UNKNOWN {
		return fmt.Errorf("validator role was not assigned for key: %x", s.pubKey)
	}

	s.role = role
	s.assignedSlot = assignedSlot
	s.shardID = shardID

	log = log.WithFields(logrus.Fields{
		"role":         role,
		"assignedSlot": assignedSlot,
		"shardID":      shardID,
	})
	return nil
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
