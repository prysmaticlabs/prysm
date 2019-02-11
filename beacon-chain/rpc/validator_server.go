package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
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
	if len(req.PublicKey) != 48 {
		return nil, fmt.Errorf("expected 48 byte public key, received %d", len(req.PublicKey))
	}
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

	for slot := req.EpochStart; slot < req.EpochStart+params.BeaconConfig().EpochLength; slot++ {
		crossLinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(beaconState, slot, false)
		if err != nil {
			return nil, err
		}
		proposerIndex, err := v.BeaconProposerIdx(beaconState, slot)
		if err != nil {
			return nil, err
		}
		if proposerIndex == validatorIndex {
			proposerSlot = slot
		}
		for _, committee := range crossLinkCommittees {
			for _, idx := range committee.Committee {
				if idx == validatorIndex {
					attesterSlot = slot
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

// ValidatorCommitteeAtSlot gets the committee at a certain slot where a validator's index is contained.
func (vs *ValidatorServer) ValidatorCommitteeAtSlot(ctx context.Context, req *pb.CommitteeRequest) (*pb.CommitteeResponse, error) {
	beaconState, err := vs.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	crossLinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(beaconState, req.Slot, false /* registry change */)
	if err != nil {
		return nil, fmt.Errorf("could not get crosslink committees at slot %d: %v", req.Slot, err)
	}
	var committee []uint64
	var shard uint64
	var indexFound bool
	for _, com := range crossLinkCommittees {
		for _, i := range com.Committee {
			if i == req.ValidatorIndex {
				committee = com.Committee
				shard = com.Shard
				indexFound = true
				break
			}
		}
		// Do not keep iterating over committees once the validator's
		// index has been found in the inner for loop.
		if indexFound {
			break
		}
	}
	return &pb.CommitteeResponse{
		Committee: committee,
		Shard:     shard,
	}, nil
}
