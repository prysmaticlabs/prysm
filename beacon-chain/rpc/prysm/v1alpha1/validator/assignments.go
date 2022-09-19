package validator

import (
	"bytes"
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/v3/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	beaconState "github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/crypto/rand"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
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
	if genesisTime.Before(prysmTime.Now()) {
		currentEpoch = slots.EpochsSinceGenesis(vs.TimeFetcher.GenesisTime())
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
	epochTicker := slots.NewSlotTicker(vs.TimeFetcher.GenesisTime(), secondsPerEpoch)
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
			currentEpoch = slots.EpochsSinceGenesis(vs.TimeFetcher.GenesisTime())
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
	currentEpoch := slots.ToEpoch(vs.TimeFetcher.CurrentSlot())
	if req.Epoch > currentEpoch+1 {
		return nil, status.Errorf(codes.Unavailable, "Request epoch %d can not be greater than next epoch %d", req.Epoch, currentEpoch+1)
	}

	s, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	// Advance state with empty transitions up to the requested epoch start slot.
	epochStartSlot, err := slots.EpochStart(req.Epoch)
	if err != nil {
		return nil, err
	}
	if s.Slot() < epochStartSlot {
		headRoot, err := vs.HeadFetcher.HeadRoot(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
		}
		s, err = transition.ProcessSlotsUsingNextSlotCache(ctx, s, headRoot, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", epochStartSlot, err)
		}
	}
	committeeAssignments, proposerIndexToSlots, err := helpers.CommitteeAssignments(ctx, s, req.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not compute committee assignments: %v", err)
	}
	// Query the next epoch assignments for committee subnet subscriptions.
	nextCommitteeAssignments, nextProposerIndexToSlots, err := helpers.CommitteeAssignments(ctx, s, req.Epoch+1)
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
			// Cache proposer assignment for the current epoch.
			for _, slot := range proposerIndexToSlots[idx] {
				// Head root is empty because it can't be known until slot - 1. Same with payload id.
				vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, idx, [8]byte{} /* payloadID */, [32]byte{} /* head root */)
			}
			// Cache proposer assignment for the next epoch.
			for _, slot := range nextProposerIndexToSlots[idx] {
				vs.ProposerSlotIndexCache.SetProposerAndPayloadIDs(slot, idx, [8]byte{} /* payloadID */, [32]byte{} /* head root */)
			}
			// Prune payload ID cache for any slots before request slot.
			vs.ProposerSlotIndexCache.PrunePayloadIDs(epochStartSlot)
		} else {
			// If the validator isn't in the beacon state, try finding their deposit to determine their status.
			vStatus, _ := vs.validatorStatus(ctx, s, pubKey)
			assignment.Status = vStatus.Status
		}

		// Are the validators in current or next epoch sync committee.
		if ok && coreTime.HigherEqualThanAltairVersionAndEpoch(s, req.Epoch) {
			assignment.IsSyncCommittee, err = helpers.IsCurrentPeriodSyncCommittee(s, idx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not determine current epoch sync committee: %v", err)
			}
			if assignment.IsSyncCommittee {
				if err := registerSyncSubnetCurrentPeriod(s, req.Epoch, pubKey, assignment.Status); err != nil {
					return nil, err
				}
			}
			nextAssignment.IsSyncCommittee = assignment.IsSyncCommittee

			// Next epoch sync committee duty is assigned with next period sync committee only during
			// sync period epoch boundary (ie. EPOCHS_PER_SYNC_COMMITTEE_PERIOD - 1). Else wise
			// next epoch sync committee duty is the same as current epoch.
			nextSlotToEpoch := slots.ToEpoch(s.Slot() + 1)
			currentEpoch := coreTime.CurrentEpoch(s)
			if slots.SyncCommitteePeriod(nextSlotToEpoch) == slots.SyncCommitteePeriod(currentEpoch)+1 {
				nextAssignment.IsSyncCommittee, err = helpers.IsNextPeriodSyncCommittee(s, idx)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "Could not determine next epoch sync committee: %v", err)
				}
				if nextAssignment.IsSyncCommittee {
					if err := registerSyncSubnetNextPeriod(s, req.Epoch, pubKey, nextAssignment.Status); err != nil {
						return nil, err
					}
				}
			}
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

// AssignValidatorToSubnet checks the status and pubkey of a particular validator
// to discern whether persistent subnets need to be registered for them.
func (vs *Server) AssignValidatorToSubnet(pubkey []byte, status ethpb.ValidatorStatus) {
	if status != ethpb.ValidatorStatus_ACTIVE && status != ethpb.ValidatorStatus_EXITING {
		return
	}

	_, ok, expTime := cache.SubnetIDs.GetPersistentSubnets(pubkey)
	if ok && expTime.After(prysmTime.Now()) {
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

func registerSyncSubnetCurrentPeriod(s beaconState.BeaconState, epoch types.Epoch, pubKey []byte, status ethpb.ValidatorStatus) error {
	committee, err := s.CurrentSyncCommittee()
	if err != nil {
		return err
	}
	syncCommPeriod := slots.SyncCommitteePeriod(epoch)
	registerSyncSubnet(epoch, syncCommPeriod, pubKey, committee, status)
	return nil
}

func registerSyncSubnetNextPeriod(s beaconState.BeaconState, epoch types.Epoch, pubKey []byte, status ethpb.ValidatorStatus) error {
	committee, err := s.NextSyncCommittee()
	if err != nil {
		return err
	}
	syncCommPeriod := slots.SyncCommitteePeriod(epoch)
	registerSyncSubnet(epoch, syncCommPeriod+1, pubKey, committee, status)
	return nil
}

// registerSyncSubnet checks the status and pubkey of a particular validator
// to discern whether persistent subnets need to be registered for them.
func registerSyncSubnet(currEpoch types.Epoch, syncPeriod uint64, pubkey []byte,
	syncCommittee *ethpb.SyncCommittee, status ethpb.ValidatorStatus) {
	if status != ethpb.ValidatorStatus_ACTIVE && status != ethpb.ValidatorStatus_EXITING {
		return
	}
	startEpoch := types.Epoch(syncPeriod * uint64(params.BeaconConfig().EpochsPerSyncCommitteePeriod))
	currPeriod := slots.SyncCommitteePeriod(currEpoch)
	endEpoch := startEpoch + params.BeaconConfig().EpochsPerSyncCommitteePeriod
	_, _, ok, expTime := cache.SyncSubnetIDs.GetSyncCommitteeSubnets(pubkey, startEpoch)
	if ok && expTime.After(prysmTime.Now()) {
		return
	}
	firstValidEpoch, err := startEpoch.SafeSub(params.BeaconConfig().SyncCommitteeSubnetCount)
	if err != nil {
		firstValidEpoch = 0
	}
	// If we are processing for a future period, we only
	// add to the relevant subscription once we are at the valid
	// bound.
	if syncPeriod != currPeriod && currEpoch < firstValidEpoch {
		return
	}
	subs := subnetsFromCommittee(pubkey, syncCommittee)
	// Handle overflow in the event current epoch is less
	// than end epoch. This is an impossible condition, so
	// it is a defensive check.
	epochsToWatch, err := endEpoch.SafeSub(uint64(currEpoch))
	if err != nil {
		epochsToWatch = 0
	}
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	totalDuration := epochDuration * time.Duration(epochsToWatch) * time.Second
	cache.SyncSubnetIDs.AddSyncCommitteeSubnets(pubkey, startEpoch, subs, totalDuration)
}

// subnetsFromCommittee retrieves the relevant subnets for the chosen validator.
func subnetsFromCommittee(pubkey []byte, comm *ethpb.SyncCommittee) []uint64 {
	positions := make([]uint64, 0)
	for i, pkey := range comm.Pubkeys {
		if bytes.Equal(pubkey, pkey) {
			positions = append(positions, uint64(i)/(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount))
		}
	}
	return positions
}
