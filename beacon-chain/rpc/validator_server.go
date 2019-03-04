package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
	index, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}

	return &pb.ValidatorIndexResponse{Index: uint64(index)}, nil
}

// CommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signalling if the validator is expected to propose a block at the assigned slot.
func (vs *ValidatorServer) CommitteeAssignment(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest) (*pb.CommitteeAssignmentResponse, error) {

	if len(req.PublicKey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf(
			"expected public key to have length %d, received %d",
			params.BeaconConfig().BLSPubkeyLength,
			len(req.PublicKey),
		)
	}

	beaconState, err := vs.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	idx, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator index: %v", err)
	}

	committee, shard, slot, isProposer, err :=
		helpers.CommitteeAssignment(beaconState, req.EpochStart, uint64(idx), false)
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

// ValidatorStatus returns the validator status of the current epoch.
// The status response can be one of the following:
//	PENDING_ACTIVE - validator is waiting to get activated.
//	ACTIVE - validator is active.
//	INITIATED_EXIT - validator has initiated an an exit request.
//	WITHDRAWABLE - validator's deposit can be withdrawn after lock up period.
//	EXITED - validator has exited, means the deposit has been withdrawn.
//	EXITED_SLASHED - validator was forcefully exited due to slashing.
func (vs *ValidatorServer) ValidatorStatus(
	ctx context.Context,
	req *pb.ValidatorIndexRequest) (*pb.ValidatorStatusResponse, error) {

	beaconState, err := vs.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	idx, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get active validator index: %v", err)
	}

	var status pb.ValidatorStatus
	v := beaconState.ValidatorRegistry[idx]
	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	epoch := helpers.CurrentEpoch(beaconState)

	if v.ActivationEpoch == farFutureEpoch {
		status = pb.ValidatorStatus_PENDING_ACTIVE
	} else if v.ActivationEpoch <= epoch && epoch < v.ExitEpoch {
		status = pb.ValidatorStatus_ACTIVE
	} else if v.InitiatedExit {
		status = pb.ValidatorStatus_INITIATED_EXIT
	} else if v.WithdrawalEpoch >= epoch {
		status = pb.ValidatorStatus_WITHDRAWABLE
	} else if epoch >= v.ExitEpoch && v.Slashed {
		status = pb.ValidatorStatus_EXITED_SLASHED
	} else if epoch >= v.ExitEpoch {
		status = pb.ValidatorStatus_EXITED
	} else {
		status = pb.ValidatorStatus_UNKNOWN_STATUS
	}

	return &pb.ValidatorStatusResponse{
		Status: status,
	}, nil
}
