package rpc

import (
	"context"
	"errors"
	"fmt"
	ptypes"github.com/gogo/protobuf/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"time"
)

type BeaconServer struct {
	beaconDB *db.BeaconDB
	ctx context.Context
	cancel context.CancelFunc
	chainService          chainService
	attestationService    attestationService
	incomingAttestation   chan *pbp2p.Attestation
	canonicalStateChan    chan *pbp2p.BeaconState
}

// CanonicalHead of the current beacon chain. This method is requested on-demand
// by a validator when it is their time to propose or attest.
func (bs *BeaconServer) CanonicalHead(ctx context.Context, req *ptypes.Empty) (*pbp2p.BeaconBlock, error) {
	block, err := bs.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get canonical head block: %v", err)
	}
	return block, nil
}

// LatestAttestation streams the latest processed attestations to the rpc clients.
func (bs *BeaconServer) LatestAttestation(req *ptypes.Empty, stream pb.BeaconService_LatestAttestationServer) error {
	sub := bs.attestationService.IncomingAttestationFeed().Subscribe(bs.incomingAttestation)
	defer sub.Unsubscribe()
	for {
		select {
		case attestation := <-bs.incomingAttestation:
			log.Info("Sending attestation to RPC clients")
			if err := stream.Send(attestation); err != nil {
				return err
			}
		case <-sub.Err():
			log.Debug("Subscriber closed, exiting goroutine")
			return nil
		case <-bs.ctx.Done():
			log.Debug("RPC context closed, exiting goroutine")
			return nil
		}
	}
}

// CurrentAssignmentsAndGenesisTime returns the current validator assignments
// based on the beacon node's current, canonical crystallized state.
// Validator clients send this request once upon establishing a connection
// to the beacon node in order to determine their role and assigned slot
// initially. This method also returns the genesis timestamp
// of the beacon node which will allow a validator client to setup a
// a ticker to keep track of the current beacon slot.
func (bs *BeaconServer) CurrentAssignmentsAndGenesisTime(
	ctx context.Context,
	req *pb.ValidatorAssignmentRequest,
) (*pb.CurrentAssignmentsResponse, error) {
	beaconState, err := bs.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	var keys []*pb.PublicKey
	if req.AllValidators {
		for _, val := range beaconState.ValidatorRegistry {
			keys = append(keys, &pb.PublicKey{PublicKey: val.Pubkey})
		}
	} else {
		keys = req.PublicKeys
		if len(keys) == 0 {
			return nil, errors.New("no public keys specified in request")
		}
	}
	assignments, err := assignmentsForPublicKeys(keys, beaconState)
	if err != nil {
		return nil, fmt.Errorf("could not get assignments for public keys: %v", err)
	}

	timestamp, err := ptypes.TimestampProto(time.Unix(int64(beaconState.GenesisTime), 0))
	if err != nil {
		return nil, fmt.Errorf("could not create timestamp proto object %v", err)
	}

	return &pb.CurrentAssignmentsResponse{
		GenesisTimestamp: timestamp,
		Assignments:      assignments,
	}, nil
}

// ValidatorAssignments streams validator assignments every cycle transition
// to clients that request to watch a subset of public keys in the
// CrystallizedState's active validator set.
func (bs *BeaconServer) ValidatorAssignments(
	req *pb.ValidatorAssignmentRequest,
	stream pb.BeaconService_ValidatorAssignmentsServer) error {

	sub := bs.chainService.CanonicalStateFeed().Subscribe(bs.canonicalStateChan)
	defer sub.Unsubscribe()
	for {
		select {
		case beaconState := <-bs.canonicalStateChan:
			log.Info("Sending new cycle assignments to validator clients")

			var keys []*pb.PublicKey
			if req.AllValidators {
				for _, val := range beaconState.ValidatorRegistry {
					keys = append(keys, &pb.PublicKey{PublicKey: val.Pubkey})
				}
			} else {
				keys = req.PublicKeys
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
		case <-bs.ctx.Done():
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
		if len(val.PublicKey) == 0 {
			continue
		}
		// For the corresponding public key and current crystallized state,
		// we determine the assigned slot for the validator and whether it
		// should act as a proposer or attester.
		assignedSlot, role, err := v.ValidatorSlotAndRole(
			val.PublicKey,
			beaconState.ValidatorRegistry,
			beaconState.ShardCommitteesAtSlots,
		)
		if err != nil {
			return nil, err
		}

		// We determine the assigned shard ID for the validator
		// based on a public key and current crystallized state.
		shardID, err := v.ValidatorShardID(
			val.PublicKey,
			beaconState.ValidatorRegistry,
			beaconState.ShardCommitteesAtSlots,
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
