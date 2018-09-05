package beacon

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/golang/protobuf/ptypes/empty"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "beacon")

type rpcClientService interface {
	BeaconServiceClient() pb.BeaconServiceClient
}

// Service that interacts with a beacon node via RPC.
type Service struct {
	ctx                    context.Context
	cancel                 context.CancelFunc
	rpcClient              rpcClientService
	validatorIndex         int
	assignedSlot           uint64
	responsibility         string
	attesterAssignmentFeed *event.Feed
	proposerAssignmentFeed *event.Feed
}

// NewBeaconValidator instantiates a service that interacts with a beacon node
// via gRPC requests.
func NewBeaconValidator(ctx context.Context, rpcClient rpcClientService) *Service {
	ctx, cancel := context.WithCancel(ctx)
	return &Service{
		ctx:                    ctx,
		cancel:                 cancel,
		rpcClient:              rpcClient,
		attesterAssignmentFeed: new(event.Feed),
		proposerAssignmentFeed: new(event.Feed),
	}
}

// Start the main routine for a beacon gRPC service.
func (s *Service) Start() {
	log.Info("Starting service")
	client := s.rpcClient.BeaconServiceClient()

	// First thing the validator does is request the genesis block timestamp
	// and the latest, canonical crystallized state from a beacon node. From here,
	// a validator can determine its assigned slot by keeping an internal
	// ticker that starts at the current slot the beacon node is in. This current slot
	// value is determined by taking the time differential between the genesis block
	// time, the latest crystallized state's time, and the current system time.
	//
	// Note: this does not validate the current system time against a global
	// NTP server, which will be important to do in production.
	// currently in a cycle we are supposed to participate in.
	// TODO: Return the ticker from here.
	s.fetchGenesisAndCanonicalState(client)

	// Then, we kick off a routine that uses the ticker set in fetchGenesisAndCanonicalState
	// to wait until the validator's assigned slot to perform proposals or attestations.
	// TODO: Pass in the ticker.
	go s.waitForAssignment()

	// We then kick off a routine that listens for streams of cycle transitions
	// coming from the beacon node. This will allow the validator client to recalculate
	// when it has to perform its responsibilities appropriately using timestamps
	// and the IndicesForSlots field inside the received crystallized state.
	go s.listenForCrystallizedStates(client)
}

// Stop the main loop..
func (s *Service) Stop() error {
	defer s.cancel()
	log.Info("Stopping service")
	return nil
}

// fetchGenesisAndCanonicalState fetches both the genesis timestamp as well
// as the latest canonical crystallized state from a beacon node. This allows
// the validator to do the following:
//
// (1) determine if it should act as an attester/proposer and at what slot
// and what shard
//
// (2) determine the difference between the genesis timestamp, latest crystallized
// state timestamp, and the current system time.
//
// From this, the validator client can deduce what slot interval the beacon
// node is in and determine when exactly it is time to propose or attest.
func (s *Service) fetchGenesisAndCanonicalState(client pb.BeaconServiceClient) {
	res, err := client.GenesisTimestampAndCanonicalState(s.ctx, &empty.Empty{})
	if err != nil {
		// If this RPC request fails, the entire system should fatal as it is critical for
		// the validator to begin this way.
		log.Fatalf("could not fetch genesis time and latest canonical state from beacon node: %v", err)
	}
	// Compute the time since genesis based on the crystallized state last_state_recalc.
	// Then, compute the difference between that value and the current system time
	// to determine what slot we are in within that cycle. Start a ticker that updates
	// this slot the runs every 8 seconds and updates this slot accordingly.
	// TODO: Implement here.
	if err := s.processCrystallizedState(res.GetLatestCrystallizedState(), client); err != nil {
		log.Fatalf("unable to process received crystallized state: %v", err)
	}
}

// waitForAssignment utilizes the computed time differential between genesis
// and current canonical crystallized state to determine which slot interval
// a beacon node is currently in. From here, a validator client can keep
// an internal ticker for checking when it is time to propose or attest
// to a block.
func (s *Service) waitForAssignment() {
	for {
		// TODO: Utilize timestamp differentials from genesis and crystallized state
		// to determine what slot interval the beacon node is currently in.
		// Using this info, then trigger the if conditions below.
		if s.responsibility == "proposer" {
			log.WithField("slotNumber", 0).Info("Assigned proposal slot number reached")
			s.responsibility = ""
			// TODO: request latest canonical block and forward it to the proposer
			// service.
		} else if s.responsibility == "attester" {
			// TODO: Let the validator know a few slots in advance if its attestation slot is coming up
			log.Info("Assigned attestation slot number reached")
			s.responsibility = ""
			// TODO: request latest canonical block and forward it to the attester
			// service.
		}
		time.Sleep(time.Second * 8)
	}
}

// listenForCrystallizedStates receives the latest canonical crystallized state
// from the beacon node's RPC server via gRPC streams.
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
		if err != nil {
			log.Errorf("Could not receive latest crystallized beacon state from stream: %v", err)
			continue
		}
		if err := s.processCrystallizedState(crystallizedState, client); err != nil {
			log.Error(err)
		}
	}
}

// processCrystallizedState uses a received crystallized state to determine
// whether a validator is a proposer/attester and the validator's assigned slot.
func (s *Service) processCrystallizedState(crystallizedState *pbp2p.CrystallizedState, client pb.BeaconServiceClient) error {
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
		// TODO: Check the public key instead of withdrawal address. This will use BLS.
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

	// TODO: Go through each of the indices for slots and determine which slot
	// a validator is assigned into and at what index.
	// indicesForSlots := res.GetIndicesForSlots()

	// The validator needs to propose the next block.
	// TODO: This is a stub until the indices for slots loop is done above.
	s.responsibility = "proposer"
	log.Debug("Validator selected as proposer of the next slot")
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

// isZeroAddress compares a withdrawal address to an empty byte array.
func isZeroAddress(withdrawalAddress []byte) bool {
	return bytes.Equal(withdrawalAddress, []byte{})
}
