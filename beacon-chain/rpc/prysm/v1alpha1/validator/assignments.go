package validator

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetDuties returns the duties assigned to a list of validators specified
// in the request object.
func (vs *Server) GetDuties(ctx context.Context, req *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	return vs.duties(ctx, req)
}

// StreamDuties returns the duties assigned to a list of validators specified
// in the request object via a server-side stream. The stream sends out new assignments in case
// a chain re-org occurred.
func (vs *Server) StreamDuties(req *ethpb.DutiesRequest, stream ethpb.BeaconNodeValidator_StreamDutiesServer) error {
	if vs.SyncChecker.Syncing() {
		return status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	// If we are post-genesis time, then set the current epoch to
	// the number epochs since the genesis time, otherwise 0 by default.
	genesisTime := vs.TimeFetcher.GenesisTime()
	if genesisTime.IsZero() {
		return status.Error(codes.Unavailable, "genesis time is not set")
	}
	var currentEpoch types.Epoch
	if genesisTime.Before(timeutils.Now()) {
		currentEpoch = slotutil.EpochsSinceGenesis(vs.TimeFetcher.GenesisTime())
	}
	req.Epoch = currentEpoch
	res, err := vs.duties(stream.Context(), req)
	if err != nil {
		return status.Errorf(codes.Internal, "Could not compute validator duties: %v", err)
	}
	if err := stream.Send(res); err != nil {
		return status.Errorf(codes.Internal, "Could not send response over stream: %v", err)
	}

	// We start a for loop which ticks on every epoch or a chain reorg.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := vs.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * uint64(params.BeaconConfig().SlotsPerEpoch)
	epochTicker := slotutil.NewSlotTicker(vs.TimeFetcher.GenesisTime(), secondsPerEpoch)
	for {
		select {
		// Ticks every epoch to submit assignments to connected validator clients.
		case slot := <-epochTicker.C():
			req.Epoch = types.Epoch(slot)
			res, err := vs.duties(stream.Context(), req)
			if err != nil {
				return status.Errorf(codes.Internal, "Could not compute validator duties: %v", err)
			}
			if err := stream.Send(res); err != nil {
				return status.Errorf(codes.Internal, "Could not send response over stream: %v", err)
			}
		case ev := <-stateChannel:
			// If a reorg occurred, we recompute duties for the connected validator clients
			// and send another response over the server stream right away.
			currentEpoch = slotutil.EpochsSinceGenesis(vs.TimeFetcher.GenesisTime())
			if ev.Type == statefeed.Reorg {
				data, ok := ev.Data.(*ethpbv1.EventChainReorg)
				if !ok {
					return status.Errorf(codes.Internal, "Received incorrect data type over reorg feed: %v", data)
				}
				req.Epoch = currentEpoch
				res, err := vs.duties(stream.Context(), req)
				if err != nil {
					return status.Errorf(codes.Internal, "Could not compute validator duties: %v", err)
				}
				if err := stream.Send(res); err != nil {
					return status.Errorf(codes.Internal, "Could not send response over stream: %v", err)
				}
			}
		case <-stream.Context().Done():
			return status.Error(codes.Canceled, "Stream context canceled")
		case <-vs.Ctx.Done():
			return status.Error(codes.Canceled, "RPC context canceled")
		}
	}
}

// Compute the validator duties from the head state's corresponding epoch
// for validators public key / indices requested.
func (vs *Server) duties(ctx context.Context, req *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	currentEpoch := helpers.SlotToEpoch(vs.TimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.Unavailable, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	epochStartSlot, err := helpers.StartSlot(req.Epoch)
	if err != nil {
		return nil, err
	}
	if s.Slot() < epochStartSlot {
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
		return nil, status.Errorf(codes.Internal, "Could not compute next committee assignments: %v", err)
	}

	validatorAssignments := make([]*ethpb.DutiesResponse_Duty, 0, len(req.PublicKeys))
	nextValidatorAssignments := make([]*ethpb.DutiesResponse_Duty, 0, len(req.PublicKeys))
	for _, pubKey := range req.PublicKeys {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, "Could not continue fetching assignments: %v", ctx.Err())
		}
		assignment := &ethpb.DutiesResponse_Duty{
			PublicKey: pubKey,
		}
		nextAssignment := &ethpb.DutiesResponse_Duty{
			PublicKey: pubKey,
		}
		idx, ok := s.ValidatorIndexByPubkey(bytesutil.ToBytes48(pubKey))
		if ok {
			s := assignmentStatus(s, idx)

			assignment.ValidatorIndex = idx
			assignment.Status = s
			assignment.ProposerSlots = proposerIndexToSlots[idx]

			// The next epoch has no lookup for proposer indexes.
			nextAssignment.ValidatorIndex = idx
			nextAssignment.Status = s

			ca, ok := committeeAssignments[idx]
			if ok {
				assignment.Committee = ca.Committee
				assignment.AttesterSlot = ca.AttesterSlot
				assignment.CommitteeIndex = ca.CommitteeIndex
			}
			// Save the next epoch assignments.
			ca, ok = nextCommitteeAssignments[idx]
			if ok {
				nextAssignment.Committee = ca.Committee
				nextAssignment.AttesterSlot = ca.AttesterSlot
				nextAssignment.CommitteeIndex = ca.CommitteeIndex
			}
		} else {
			// If the validator isn't in the beacon state, try finding their deposit to determine their status.
			vStatus, _ := vs.validatorStatus(ctx, s, pubKey)
			assignment.Status = vStatus.Status
		}
		validatorAssignments = append(validatorAssignments, assignment)
		nextValidatorAssignments = append(nextValidatorAssignments, nextAssignment)
		// Assign relevant validator to subnet.
		vs.AssignValidatorToSubnet(pubKey, assignment.Status)
		vs.AssignValidatorToSubnet(pubKey, nextAssignment.Status)
	}

	return &ethpb.DutiesResponse{
		Duties:             validatorAssignments,
		CurrentEpochDuties: validatorAssignments,
		NextEpochDuties:    nextValidatorAssignments,
	}, nil
}

// TODO testy
// AssignValidatorToSubnet checks the status and pubkey of a particular validator
// to discern whether persistent subnets need to be registered for them.
func (vs *Server) AssignValidatorToSubnet(pubkey []byte, status ethpb.ValidatorStatus) {
	if status != ethpb.ValidatorStatus_ACTIVE && status != ethpb.ValidatorStatus_EXITING {
		return
	}

	_, ok, expTime := cache.SubnetIDs.GetPersistentSubnets(pubkey)
	if ok && expTime.After(timeutils.Now()) {
		return
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	var assignedIdxs []uint64
	randGen := rand.NewGenerator()
	for i := uint64(0); i < params.BeaconConfig().RandomSubnetsPerValidator; i++ {
		assignedIdx := randGen.Intn(int(params.BeaconNetworkConfig().AttestationSubnetCount))
		assignedIdxs = append(assignedIdxs, uint64(assignedIdx))
	}

	assignedDuration := uint64(randGen.Intn(int(params.BeaconConfig().EpochsPerRandomSubnetSubscription)))
	assignedDuration += params.BeaconConfig().EpochsPerRandomSubnetSubscription

	totalDuration := epochDuration * time.Duration(assignedDuration)
	cache.SubnetIDs.AddPersistentCommittee(pubkey, assignedIdxs, totalDuration*time.Second)
}
