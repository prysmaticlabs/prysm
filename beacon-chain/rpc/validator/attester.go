package validator

import (
	"context"
	"errors"
	"fmt"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetAttestationData requests that the beacon node produce an attestation data object,
// which the validator acting as an attester will then sign.
func (vs *Server) GetAttestationData(ctx context.Context, req *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.RequestAttestation")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("slot", int64(req.Slot)),
		trace.Int64Attribute("committeeIndex", int64(req.CommitteeIndex)),
	)

	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	if err := helpers.ValidateAttestationTime(req.Slot, vs.GenesisTimeFetcher.GenesisTime()); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request: %v", err))
	}

	// Attester will either wait until there's a valid block from the expected block proposer of for the assigned input slot
	// or one third of the slot has transpired. Whichever comes first.
	vs.waitOneThirdOrValidBlock(ctx, req.Slot)

	res, err := vs.AttestationCache.Get(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve data from attestation cache: %v", err)
	}
	if res != nil {
		res.CommitteeIndex = req.CommitteeIndex
		return res, nil
	}

	if err := vs.AttestationCache.MarkInProgress(req); err != nil {
		if errors.Is(err, cache.ErrAlreadyInProgress) {
			res, err := vs.AttestationCache.Get(ctx, req)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve data from attestation cache: %v", err)
			}
			if res == nil {
				return nil, status.Error(codes.DataLoss, "A request was in progress and resolved to nil")
			}
			res.CommitteeIndex = req.CommitteeIndex
			return res, nil
		}
		return nil, status.Errorf(codes.Internal, "Could not mark attestation as in-progress: %v", err)
	}
	defer func() {
		if err := vs.AttestationCache.MarkNotInProgress(req); err != nil {
			log.WithError(err).Error("Could not mark cache not in progress")
		}
	}()

	headState, err := vs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head state: %v", err)
	}
	headRoot, err := vs.HeadFetcher.HeadRoot(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head root: %v", err)
	}

	// In the case that we receive an attestation request after a newer state/block has been processed.
	if headState.Slot() > req.Slot {
		headRoot, err = helpers.BlockRootAtSlot(headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get historical head root: %v", err)
		}
		headState, err = vs.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(headRoot))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get historical head state: %v", err)
		}
	}
	if headState == nil {
		return nil, status.Error(codes.Internal, "Could not lookup parent state from head.")
	}

	if helpers.CurrentEpoch(headState) < helpers.SlotToEpoch(req.Slot) {
		headState, err = state.ProcessSlots(ctx, headState, req.Slot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
		}
	}

	targetEpoch := helpers.CurrentEpoch(headState)
	epochStartSlot, err := helpers.StartSlot(targetEpoch)
	if err != nil {
		return nil, err
	}
	var targetRoot []byte
	if epochStartSlot == headState.Slot() {
		targetRoot = headRoot
	} else {
		targetRoot, err = helpers.BlockRootAtSlot(headState, epochStartSlot)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not get target block for slot %d: %v", epochStartSlot, err)
		}
		if bytesutil.ToBytes32(targetRoot) == params.BeaconConfig().ZeroHash {
			targetRoot = headRoot
		}
	}

	res = &ethpb.AttestationData{
		Slot:            req.Slot,
		CommitteeIndex:  req.CommitteeIndex,
		BeaconBlockRoot: headRoot,
		Source:          headState.CurrentJustifiedCheckpoint(),
		Target: &ethpb.Checkpoint{
			Epoch: targetEpoch,
			Root:  targetRoot,
		},
	}

	if err := vs.AttestationCache.Put(ctx, req, res); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not store attestation data in cache: %v", err)
	}
	return res, nil
}

// ProposeAttestation is a function called by an attester to vote
// on a block via an attestation object as defined in the Ethereum Serenity specification.
func (vs *Server) ProposeAttestation(ctx context.Context, att *ethpb.Attestation) (*ethpb.AttestResponse, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.ProposeAttestation")
	defer span.End()

	if _, err := bls.SignatureFromBytes(att.Signature); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Incorrect attestation signature")
	}

	root, err := att.Data.HashTreeRoot()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not tree hash attestation: %v", err)
	}

	// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
	// of a received unaggregated attestation.
	vs.OperationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.UnaggregatedAttReceived,
		Data: &operation.UnAggregatedAttReceivedData{
			Attestation: att,
		},
	})

	// Determine subnet to broadcast attestation to
	wantedEpoch := helpers.SlotToEpoch(att.Data.Slot)
	vals, err := vs.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
	if err != nil {
		return nil, err
	}
	subnet := helpers.ComputeSubnetFromCommitteeAndSlot(uint64(len(vals)), att.Data.CommitteeIndex, att.Data.Slot)

	// Broadcast the new attestation to the network.
	if err := vs.P2P.BroadcastAttestation(ctx, subnet, att); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast attestation: %v", err)
	}

	go func() {
		ctx = trace.NewContext(context.Background(), trace.FromContext(ctx))
		attCopy := stateTrie.CopyAttestation(att)
		if err := vs.AttPool.SaveUnaggregatedAttestation(attCopy); err != nil {
			log.WithError(err).Error("Could not handle attestation in operations service")
			return
		}
	}()

	return &ethpb.AttestResponse{
		AttestationDataRoot: root[:],
	}, nil
}

// SubscribeCommitteeSubnets subscribes to the committee ID subnet given subscribe request.
func (vs *Server) SubscribeCommitteeSubnets(ctx context.Context, req *ethpb.CommitteeSubnetsSubscribeRequest) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.SubscribeCommitteeSubnets")
	defer span.End()

	if len(req.Slots) != len(req.CommitteeIds) || len(req.CommitteeIds) != len(req.IsAggregator) {
		return nil, status.Error(codes.InvalidArgument, "request fields are not the same length")
	}
	if len(req.Slots) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no attester slots provided")
	}

	fetchValsLen := func(slot uint64) (uint64, error) {
		wantedEpoch := helpers.SlotToEpoch(slot)
		vals, err := vs.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			return 0, err
		}
		return uint64(len(vals)), nil
	}

	// Request the head validator indices of epoch represented by the first requested
	// slot.
	currValsLen, err := fetchValsLen(req.Slots[0])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve head validator length: %v", err)
	}
	currEpoch := helpers.SlotToEpoch(req.Slots[0])

	for i := 0; i < len(req.Slots); i++ {
		// If epoch has changed, re-request active validators length
		if currEpoch != helpers.SlotToEpoch(req.Slots[i]) {
			currValsLen, err = fetchValsLen(req.Slots[i])
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve head validator length: %v", err)
			}
			currEpoch = helpers.SlotToEpoch(req.Slots[i])
		}
		subnet := helpers.ComputeSubnetFromCommitteeAndSlot(currValsLen, req.CommitteeIds[i], req.Slots[i])
		cache.SubnetIDs.AddAttesterSubnetID(req.Slots[i], subnet)
		if req.IsAggregator[i] {
			cache.SubnetIDs.AddAggregatorSubnetID(req.Slots[i], subnet)
		}
	}

	return &ptypes.Empty{}, nil
}

// waitOneThirdOrValidBlock waits until (a) or (b) whichever comes first:
//   (a) the validator has received a valid block from the expected block proposer for the assigned slot
//   (b) one-third of the slot has transpired (SECONDS_PER_SLOT / 3 seconds after the start of slot)
func (vs *Server) waitOneThirdOrValidBlock(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToOneThird")
	defer span.End()

	// Don't need to wait if current slot is greater than requested slot.
	if slot < vs.GenesisTimeFetcher.CurrentSlot() {
		return
	}

	// Set time out to be at start slot time + one-third of slot duration.
	slotStartTime := slotutil.SlotStartTime(uint64(vs.GenesisTimeFetcher.GenesisTime().Unix()), slot)
	slotOneThirdTime := slotStartTime.Unix() + int64(params.BeaconConfig().SecondsPerSlot/3)
	waitDuration := slotOneThirdTime - time.Now().Unix()
	timeOut := time.After(time.Duration(waitDuration) * time.Second)

	stateChannel := make(chan *feed.Event, 1)
	stateSub := vs.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	for {
		select {
		case event := <-stateChannel:
			// Node processed a block, check if the processed block is the same as input slot.
			if event.Type == statefeed.BlockProcessed {
				d, ok := event.Data.(*statefeed.BlockProcessedData)
				if !ok {
					log.Error("Event feed data is not type *statefeed.BlockProcessedData")
					continue
				}
				if slot == d.Slot {
					return
				}
			}
		case <-timeOut:
			return
		case <-ctx.Done():
			return
		case err := <-stateSub.Err():
			log.WithError(err).Error("Subscription to state notifier failed at wait to one third")
			return
		}
	}
}
