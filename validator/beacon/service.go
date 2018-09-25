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
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
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
	validatorClient          validatorClientService
	validatorIndex           int
	assignedSlot             uint64
	assignedShardID          uint64
	responsibility           pb.ValidatorRole
	attesterAssignmentFeed   *event.Feed
	proposerAssignmentFeed   *event.Feed
	processedAttestationFeed *event.Feed
	genesisTimestamp         time.Time
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
	}
}

// Start the main routine for a beacon client service.
func (s *Service) Start() {
	log.Info("Starting service")
	client := s.rpcClient.BeaconServiceClient()
	validator := s.validatorClient.ValidatorServiceClient()

	// First thing the validator does is request the genesis block timestamp
	// and the latest, canonical crystallized state from a beacon node. From here,
	// a validator can determine its assigned slot by keeping an internal
	// ticker that starts at the current slot the beacon node is in. This current slot
	// value is determined by taking the time differential between the genesis block
	// time and the current system time.
	//
	// Note: this does not validate the current system time against a global
	// NTP server, which will be important to do in production.
	// currently in a cycle we are supposed to participate in.
	s.fetchGenesisAndCanonicalState(client)

	// We kick off a routine that listens for stream of validator assignment coming from
	// beacon node. This will update validator client on which slot, shard ID and what
	// responsbility to perform.
	go s.listenForAssignmentChange(validator)

	// Then, we kick off a routine that uses the begins a ticker set in fetchGenesisAndCanonicalState
	// to wait until the validator's assigned slot to perform proposals or attestations.
	ticker := time.NewTicker(time.Second)
	go s.waitForAssignment(ticker.C, client)

	// Finally, we then kick off a routine that listens for streams of cycle transitions
	// coming from the beacon node. This will allow the validator client to recalculate
	// when it has to perform its responsibilities appropriately using timestamps
	// and the IndicesForSlots field inside the received crystallized state.
	go s.listenForCrystallizedStates(client)
	go s.listenForProcessedAttestations(client)
}

// Stop the main loop..
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// CurrentBeaconSlot based on the seconds since genesis.
func (s *Service) CurrentBeaconSlot() uint64 {
	secondsSinceGenesis := time.Since(s.genesisTimestamp).Seconds()
	return uint64(math.Floor(secondsSinceGenesis / 8.0))
}

// fetchGenesisAndCanonicalState fetches both the genesis timestamp as well
// as the latest canonical crystallized state from a beacon node. This allows
// the validator to do the following:
//
// (1) determine if it should act as an attester/proposer and at what slot
// and what shard
//
// (2) determine the seconds since genesis by using the latest crystallized
// state recalc, then determine how many seconds have passed between that time
// and the current system time.
//
// From this, the validator client can deduce what slot interval the beacon
// node is in and determine when exactly it is time to propose or attest.
func (s *Service) fetchGenesisAndCanonicalState(client pb.BeaconServiceClient) {
	res, err := client.GenesisTimeAndCanonicalState(s.ctx, &empty.Empty{})
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

	crystallized := res.GetLatestCrystallizedState()
	if err := s.processCrystallizedState(crystallized); err != nil {
		log.Fatalf("unable to process received crystallized state: %v", err)
	}
}

func (s *Service) listenForAssignmentChange(validator pb.ValidatorServiceClient) {
	req := &pb.ValidatorAssignmentRequest{PublicKeys: []*pb.PublicKey{{PublicKey: s.pubKey}}}
	stream, err := validator.ValidatorAssignment(s.ctx, req)
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
		s.responsibility = assignment.Assignments[0].Role
		s.assignedSlot = assignment.Assignments[0].AssignedSlot
		s.assignedShardID = assignment.Assignments[0].ShardId

		log.Infof("Validator with pub key 0x%v re-assigned to shard ID %d for %v duty at slot %d",
			string(assignment.Assignments[0].PublicKey.PublicKey),
			s.assignedShardID,
			s.responsibility,
			s.assignedSlot)
	}
}

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
		}
	}
}

// listenForCrystallizedStates receives the latest canonical crystallized state
// from the beacon node's RPC server via gRPC streams.
// TODO(#545): Rename to listen for assignment instead, which is streamed from a beacon node
// upon every new cycle transition and will include the validator's index in the
// assignment bitfield as well as the assigned shard ID.
func (s *Service) listenForCrystallizedStates(client pb.BeaconServiceClient) {
	stream, err := client.LatestCrystallizedState(s.ctx, &empty.Empty{})
	if err != nil {
		log.Errorf("Could not setup crystallized beacon state streaming client: %v", err)
		return
	}
	for {
		crystallizedState, err := stream.Recv()
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
			log.Errorf("Could not receive latest crystallized beacon state from stream: %v", err)
			continue
		}
		if err := s.processCrystallizedState(crystallizedState); err != nil {
			log.Error(err)
		}
	}
}

// processCrystallizedState uses a received crystallized state to determine
// whether a validator is a proposer/attester and the validator's assigned slot.
func (s *Service) processCrystallizedState(crystallizedState *pbp2p.CrystallizedState) error {
	var activeValidatorIndices []int
	dynasty := crystallizedState.GetCurrentDynasty()

	for i, validator := range crystallizedState.GetValidators() {
		if validator.StartDynasty <= dynasty && dynasty < validator.EndDynasty {
			activeValidatorIndices = append(activeValidatorIndices, i)
		}
	}
	isValidatorIndexSet := false

	// We then iteratate over the activeValidatorIndices to determine what index
	// this running validator client corresponds to.
	for _, val := range activeValidatorIndices {
		// TODO(#258): Check the public key instead of withdrawal address. This will use BLS.
		if isZeroAddress(crystallizedState.Validators[val].WithdrawalAddress) {
			s.validatorIndex = val
			isValidatorIndexSet = true
			break
		}
	}

	// If validator was not found in the validator set was not set, keep listening for
	// crystallized states.
	if !isValidatorIndexSet {
		log.Debug("Validator index not found in latest crystallized state's active validator list")
		return nil
	}

	// The validator needs to propose the next block.
	// TODO(#545): Determine this from a gRPC stream from the beacon node
	// instead.
	s.responsibility = pb.ValidatorRole_PROPOSER
	s.assignedSlot = s.CurrentBeaconSlot() + 2
	log.WithField("assignedSlot", s.assignedSlot).Info("Validator selected as proposer")
	return nil
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

// isZeroAddress compares a withdrawal address to an empty byte array.
func isZeroAddress(withdrawalAddress []byte) bool {
	return bytes.Equal(withdrawalAddress, []byte{})
}
