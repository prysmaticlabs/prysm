package rpc

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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

// ValidatorEpochAssignments fetches an assignment object for a validator by public key
// such as the slot the validator needs to attest in during the epoch as well as a slot
// in which the validator may need to propose during the epoch in addition to the assigned shard.
func (vs *ValidatorServer) ValidatorEpochAssignments(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest,
) (*pb.ValidatorEpochAssignmentsResponse, error) {
	if len(req.PublicKey) != params.BeaconConfig().BLSPubkeyLength {
		return nil, fmt.Errorf(
			"expected public key to have length %d, received %d",
			params.BeaconConfig().BLSPubkeyLength,
			len(req.PublicKey),
		)
	}
	beaconState, err := vs.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get beacon state: %v", err)
	}
	head, err := vs.beaconDB.ChainHead()
	if err != nil {
		return nil, fmt.Errorf("could not get chain head: %v", err)
	}
	headRoot := bytesutil.ToBytes32(head.ParentRootHash32)
	beaconState, err = state.ExecuteStateTransition(
		beaconState, nil /* block */, headRoot, false, /* verify signatures */
	)
	if err != nil {
		return nil, fmt.Errorf("could not execute head transition: %v", err)
	}
	validatorIndex, err := vs.beaconDB.ValidatorIndex(req.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not get validator index: %v", err)
	}
	var shard uint64
	var attesterSlot uint64
	var proposerSlot uint64

	for slot := req.EpochStart; slot < req.EpochStart+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var registryChanged bool
		if beaconState.ValidatorRegistryUpdateEpoch == helpers.SlotToEpoch(slot)-1 &&
			beaconState.ValidatorRegistryUpdateEpoch != params.BeaconConfig().GenesisEpoch {
			registryChanged = true
		}
		crossLinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(beaconState, slot, registryChanged /* registry change */)
		if err != nil {
			return nil, fmt.Errorf("could not get crosslink committees at slot %d: %v", slot-params.BeaconConfig().GenesisSlot, err)
		}
		proposerIndex, err := helpers.BeaconProposerIndex(beaconState, slot)
		if err != nil {
			return nil, err
		}
		log.Debugf("Proposer index: %d, slot: %d", proposerIndex, slot-params.BeaconConfig().GenesisSlot)
		if proposerIndex == uint64(validatorIndex) {
			proposerSlot = slot
		}
		for _, committee := range crossLinkCommittees {
			for _, idx := range committee.Committee {
				if idx == uint64(validatorIndex) {
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
	beaconState, err := vs.beaconDB.State(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not fetch beacon state: %v", err)
	}
	if req.Slot%params.BeaconConfig().SlotsPerEpoch == 0 {
		head, err := vs.beaconDB.ChainHead()
		if err != nil {
			return nil, fmt.Errorf("could not get chain head: %v", err)
		}
		headRoot := bytesutil.ToBytes32(head.ParentRootHash32)
		beaconState, err = state.ExecuteStateTransition(
			beaconState, nil /* block */, headRoot, false, /* verify signatures */
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute head transition: %v", err)
		}
	}
	var registryChanged bool
	if beaconState.ValidatorRegistryUpdateEpoch == helpers.SlotToEpoch(req.Slot)-1 &&
		beaconState.ValidatorRegistryUpdateEpoch != params.BeaconConfig().GenesisEpoch {
		registryChanged = true
	}
	crossLinkCommittees, err := helpers.CrosslinkCommitteesAtSlot(beaconState, req.Slot, registryChanged /* registry change */)
	if err != nil {
		return nil, fmt.Errorf("could not get crosslink committees at slot %d: %v", req.Slot-params.BeaconConfig().GenesisSlot, err)
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

// CommitteeAssignment returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signalling if the validator is expected to propose a block at the assigned slot.
func (vs *ValidatorServer) CommitteeAssignment(
	ctx context.Context,
	req *pb.ValidatorEpochAssignmentsRequest) (*pb.CommitteeAssignmentResponse, error) {

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
	} else if v.StatusFlags == pbp2p.Validator_INITIATED_EXIT {
		status = pb.ValidatorStatus_INITIATED_EXIT
	} else if v.StatusFlags == pbp2p.Validator_WITHDRAWABLE {
		status = pb.ValidatorStatus_WITHDRAWABLE
	} else if epoch >= v.ExitEpoch && epoch >= v.SlashedEpoch {
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
