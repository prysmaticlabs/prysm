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

// NextEpochCommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the next epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signalling if the validator is expected to propose a block at the assigned slot.
func (vs *ValidatorServer) NextEpochCommitteeAssignment(
	ctx context.Context,
	req *pb.ValidatorIndexRequest) (*pb.CommitteeAssignmentResponse, error) {

	if len(req.PublicKey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf(
			"expected public key to have length %d, received %d",
			params.BeaconConfig().BLSPubkeyLength,
			len(req.PublicKey),
		)
	}

	state, err := vs.beaconDB.State()
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	idx, err := v.ValidatorIdx(req.PublicKey, state.ValidatorRegistry)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator index: %v", err)
	}

	committee, shard, slot, isProposer, err :=
		helpers.NextEpochCommitteeAssignment(state, idx, false)
	if err != nil {
		return nil, fmt.Errorf("could not get next epoch committee assignment: %v", err)
	}

	return &pb.CommitteeAssignmentResponse{
		Committee:  committee,
		Shard:      shard,
		Slot:       slot,
		IsProposer: isProposer,
	}, nil
}
