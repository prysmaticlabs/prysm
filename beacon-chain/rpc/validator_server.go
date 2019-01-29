package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/params"

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

// ValidatorIndex is called by a validator to get its index location that corresponds
// to the attestation bit fields.
func (vs *ValidatorServer) ValidatorIndex(ctx context.Context, req *pb.ValidatorIndexRequest) (*pb.ValidatorIndexResponse, error) {
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

// ValidatorEpochAssignments fetches an assignment object for a validator by public key
// such as the slot the validator needs to attest in during the epoch as well as a slot
// in which the validator may need to propose during the epoch in addition to the assigned shard.
func (vs *ValidatorServer) ValidatorEpochAssignments(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest,
) (*pb.ValidatorEpochAssignmentsResponse, error) {
	beaconState, err := vs.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	validatorIndex, err := v.ValidatorIdx(req.PublicKey, beaconState.ValidatorRegistry)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator index: %v", err)
	}
	var shard uint64
	var attesterSlot uint64
	var proposerSlot uint64

	for i := req.EpochStart; i < req.EpochStart+params.BeaconConfig().EpochLength; i++ {
		crossLinkCommittees, err := v.CrosslinkCommitteesAtSlot(beaconState, i)
		if err != nil {
			return nil, fmt.Errorf("could not get crosslink committees at slot %d: %v", i, err)
		}
		firstCommittee := crossLinkCommittees[0].Committee
		proposerIndex := firstCommittee[i%uint64(len(firstCommittee))]
		if proposerIndex == validatorIndex {
			proposerSlot = i
		}
		for _, committee := range crossLinkCommittees {
			for _, idx := range committee.Committee {
				if idx == validatorIndex {
					attesterSlot = i
					shard = committee.Shard
				}
			}
		}
	}
	return &pb.ValidatorEpochAssignmentsResponse{
		Assignment: &pb.Assignment{
			PublicKey:    req.PublicKey,
			Shard:        shard,
			AttesterSlot: attesterSlot,
			ProposerSlot: proposerSlot,
		},
	}, nil
}
