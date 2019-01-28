package rpc

import (
	"context"
	"fmt"

	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
)

// ValidatorServer defines a server implementation of the gRPC Validator service,
// providing RPC endpoints for obtaining validator assignments per epoch, the slots
// and shards in which particular validators need to perform their responsibilities,
// and more.
type ValidatorServer struct {
	beaconDB *db.BeaconDB
}

// ValidatorShardID is called by a validator to get the shard ID of where it's suppose
// to proposer or attest.
func (vs *ValidatorServer) ValidatorShardID(ctx context.Context, req *pb.PublicKey) (*pb.ShardIDResponse, error) {
	beaconState, err := vs.beaconDB.State()
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
func (vs *ValidatorServer) ValidatorSlotAndResponsibility(
	ctx context.Context,
	req *pb.PublicKey,
) (*pb.SlotResponsibilityResponse, error) {
	beaconState, err := vs.beaconDB.State()
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
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.PublicKey) (*pb.IndexResponse, error) {
	beaconState, err := vs.beaconDB.State()
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
func (vs *ValidatorServer) ValidatorEpochAssignments(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest,
) (*pb.ValidatorEpochAssignmentsResponse, error) {
	return nil, nil
}
