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

// ValidatorShard is called by a validator to get the shard of where it needs to act
// as a proposer or attester based on the current active validator registry.
func (vs *ValidatorServer) ValidatorShard(ctx context.Context, req *pb.PublicKey) (*pb.ValidatorShardResponse, error) {
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

	return &pb.ValidatorShardResponse{Shard: shardID}, nil
}

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.PublicKey) (*pb.ValidatorIndexResponse, error) {
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

	return &pb.ValidatorIndexResponse{Index: index}, nil
}

// ValidatorEpochAssignments ... WIP
func (vs *ValidatorServer) ValidatorEpochAssignments(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest,
) (*pb.ValidatorEpochAssignmentsResponse, error) {
	return nil, nil
}
