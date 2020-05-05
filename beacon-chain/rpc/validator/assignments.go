package validator

import (
	"context"
	"math/rand"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDuties returns the committee assignment response from a given validator public key.
// The committee assignment response contains the following fields for the current and previous epoch:
//	1.) The list of validators in the committee.
//	2.) The shard to which the committee is assigned.
//	3.) The slot at which the committee is assigned.
//	4.) The bool signaling if the validator is expected to propose a block at the assigned slot.
func (vs *Server) GetDuties(ctx context.Context, req *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	if epochStartSlot := helpers.StartSlot(req.Epoch); s.Slot() < epochStartSlot {
		s, err = state.ProcessSlots(ctx, s, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}
	// Query the next epoch assignments for committee subnet subscriptions.
	nextCommitteeAssignments, _, err := helpers.CommitteeAssignments(s, req.Epoch+1)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}

	var committeeIDs []uint64
	var nextCommitteeIDs []uint64
	validatorAssignments := make([]*ethpb.DutiesResponse_Duty, 0, len(req.PublicKeys))
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, "Could not continue fetching assignments: %v", ctx.Err())
		}
		// Default assignment.
		assignment := &ethpb.DutiesResponse_Duty{
			PublicKey: pubKey,
		}

		idx, ok := s.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if ok {
			assignment.ValidatorIndex = idx
			assignment.Status = vs.assignmentStatus(idx, s)
			assignment.ProposerSlots = proposerIndexToSlots[idx]

			ca, ok := committeeAssignments[idx]
			if ok {
				assignment.Committee = ca.Committee
				assignment.AttesterSlot = ca.AttesterSlot
				assignment.CommitteeIndex = ca.CommitteeIndex
				committeeIDs = append(committeeIDs, ca.CommitteeIndex)
			}
			// Save the next epoch assignments.
			ca, ok = nextCommitteeAssignments[idx]
			if ok {
				nextCommitteeIDs = append(nextCommitteeIDs, ca.CommitteeIndex)
			}
			// Assign relevant validator to subnet.
			assignValidatorToSubnet(pubKey, assignment.Status)

		} else {
			// If the validator isn't in the beacon state, assume their status is unknown.
			assignment.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		}
		validatorAssignments = append(validatorAssignments, assignment)
	}

	return &ethpb.DutiesResponse{
		Duties: validatorAssignments,
	}, nil
}

// StreamDuties --
func (vs *Server) StreamDuties(stream ethpb.BeaconNodeValidator_StreamDutiesServer) error {
	return status.Error(codes.Unimplemented, "unimplemented")
}

// assignValidatorToSubnet checks the status and pubkey of a particular validator
// to discern whether persistent subnets need to be registered for them.
func assignValidatorToSubnet(pubkey []byte, status ethpb.ValidatorStatus) {
	if status != ethpb.ValidatorStatus_ACTIVE && status != ethpb.ValidatorStatus_EXITING {
		return
	}

	_, ok, expTime := cache.CommitteeIDs.GetPersistentCommittees(pubkey)
	if ok && expTime.After(roughtime.Now()) {
		return
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch * params.BeaconConfig().SecondsPerSlot)
	assignedIdxs := []uint64{}
	for i := uint64(0); i < params.BeaconNetworkConfig().RandomSubnetsPerValidator; i++ {
		assignedIndex := rand.Intn(int(params.BeaconNetworkConfig().AttestationSubnetCount))
		assignedIdxs = append(assignedIdxs, uint64(assignedIndex))
	}
	assignedDuration := rand.Intn(int(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription))
	assignedDuration += int(params.BeaconNetworkConfig().EpochsPerRandomSubnetSubscription)

	totalDuration := epochDuration * time.Duration(assignedDuration)
	cache.CommitteeIDs.AddPersistentCommittee(pubkey, assignedIdxs, totalDuration*time.Second)
}
