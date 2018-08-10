package actor

import (
	"bytes"
	"context"
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/prysmaticlabs/prysm/client/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

var log = logrus.WithField("prefix", "actor")

// Actor stuff.
type Actor struct {
	ctx              context.Context
	cancel           context.CancelFunc
	clientService    types.RPCClient
	validatorIndex   int
	assignedHeight   uint64
	isHeightAssigned bool
	attesterChan     chan bool
	proposerChan     chan bool
}

// Config options for the actor service.
type Config struct {
	AttesterChanBuf int
	ProposerChanBuf int
}

// NewActor hi.
func NewActor(ctx context.Context, cfg *Config, clientService types.RPCClient) *Actor {
	ctx, cancel := context.WithCancel(ctx)
	return &Actor{
		ctx:           ctx,
		cancel:        cancel,
		clientService: clientService,
		attesterChan:  make(chan bool, cfg.AttesterChanBuf),
		proposerChan:  make(chan bool, cfg.ProposerChanBuf),
	}
}

// Start the main routine for an actor.
func (act *Actor) Start() {
	log.Info("Starting service")
	rpcClient := act.clientService.BeaconServiceClient()
	go act.fetchBeaconBlocks(rpcClient)
	go act.fetchCrystallizedState(rpcClient)
}

// Stop the main loop..
func (act *Actor) Stop() error {
	defer act.cancel()
	log.Info("Stopping service")
	return nil
}

func (act *Actor) fetchBeaconBlocks(client pb.BeaconServiceClient) {
	stream, err := client.LatestBeaconBlock(act.ctx, &empty.Empty{})
	if err != nil {
		log.Errorf("Could not setup beacon chain block streaming client: %v", err)
		return
	}
	for {
		block, err := stream.Recv()

		// If the stream is closed, we stop the loop.
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Errorf("Could not receive latest beacon block from stream: %v", err)
			continue
		}
		log.WithField("slotNumber", block.GetSlotNumber()).Info("Latest beacon block slot number")

		// Based on the height determined from the latest crystallized state, check if
		// it matches the latest received beacon height.
		if act.isHeightAssigned && block.GetSlotNumber() == act.assignedHeight {
			// Reset is height assigned.
			act.isHeightAssigned = false
			// TODO: perform responsibility.
		}
	}
}

func (act *Actor) fetchCrystallizedState(client pb.BeaconServiceClient) {
	stream, err := client.LatestCrystallizedState(act.ctx, &empty.Empty{})
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
		// After receiving the crystallized state, get its hash, and
		// this attester's index in the list.
		stateData, err := proto.Marshal(crystallizedState)
		if err != nil {
			log.Errorf("Could not marshal crystallized state proto: %v", err)
			continue
		}
		crystallizedStateHash := blake2b.Sum256(stateData)

		activeValidators := crystallizedState.GetActiveValidators()

		isValidatorIndexSet := false

		for i, val := range activeValidators {
			// TODO: Check the public key instead of withdrawal address. This will
			// use BLS.
			if isZeroAddress(val.GetWithdrawalAddress()) {
				act.validatorIndex = i
				isValidatorIndexSet = true
				break
			}
		}

		// If validator was not found in the validator set was not set, keep listening for
		// crystallized states.
		if !isValidatorIndexSet {
			log.Debug("Validator index not found in latest crystallized state's active validator list")
			continue
		}

		req := &pb.ShuffleRequest{
			CrystallizedStateHash: crystallizedStateHash[:],
		}

		res, err := client.FetchShuffledValidatorIndices(act.ctx, req)
		if err != nil {
			log.Errorf("Could not fetch shuffled validator indices: %v", err)
			continue
		}

		log.Info(res)
	}
}

// isZeroAddress compares a withdrawal address to an empty byte array.
func isZeroAddress(withdrawalAddress []byte) bool {
	return bytes.Equal(withdrawalAddress, []byte{})
}
