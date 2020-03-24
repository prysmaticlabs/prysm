package validator

import (
	"context"
	"fmt"
	"time"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const msgInvalidAttestationRequest = "Attestation request must be within current or previous epoch"

// GetAttestationData requests that the beacon node produce an attestation data object,
// which the validator acting as an attester will then sign.
func (vs *Server) GetAttestationData(ctx context.Context, req *ethpb.AttestationDataRequest) (*ethpb.AttestationData, error) {
	ctx, span := trace.StartSpan(ctx, "AttesterServer.RequestAttestation")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("slot", int64(req.Slot)),
		trace.Int64Attribute("committeeIndex", int64(req.CommitteeIndex)),
	)

	// If attestation committee subnets are enabled, we track the committee
	// index into a cache.
	if featureconfig.Get().EnableDynamicCommitteeSubnets {
		cache.CommitteeIDs.AddIDs([]uint64{req.CommitteeIndex}, helpers.SlotToEpoch(req.Slot))
	}

	if vs.SyncChecker.Syncing() {
		return nil, status.Errorf(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}

	currentEpoch := helpers.SlotToEpoch(helpers.SlotsSince(vs.GenesisTime))
	if currentEpoch-1 != helpers.SlotToEpoch(req.Slot) && currentEpoch != helpers.SlotToEpoch(req.Slot) {
		return nil, status.Error(codes.InvalidArgument, msgInvalidAttestationRequest)
	}

	// Attester will either wait until there's a valid block from the expected block proposer of for the assigned input slot
	// or one third of the slot has transpired. Whichever comes first.
	vs.waitToOneThird(ctx, req.Slot)

	res, err := vs.AttestationCache.Get(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve data from attestation cache: %v", err)
	}
	if res != nil {
		return res, nil
	}

	if err := vs.AttestationCache.MarkInProgress(req); err != nil {
		if err == cache.ErrAlreadyInProgress {
			res, err := vs.AttestationCache.Get(ctx, req)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Could not retrieve data from attestation cache: %v", err)
			}
			if res == nil {
				return nil, status.Error(codes.DataLoss, "A request was in progress and resolved to nil")
			}
			return res, nil
		}
		return nil, status.Errorf(codes.Internal, "Could not mark attestation as in-progress: %v", err)
	}
	defer func() {
		if err := vs.AttestationCache.MarkNotInProgress(req); err != nil {
			log.WithError(err).Error("Failed to mark cache not in progress")
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

	// In the case that we receive an attestation request after a newer state/block has been
	// processed, we walk up the chain until state.Slot <= req.Slot to prevent producing an
	// attestation that violates processing constraints.
	for headState.Slot() > req.Slot {
		if ctx.Err() != nil {
			return nil, status.Errorf(codes.Aborted, ctx.Err().Error())
		}
		parent := headState.ParentRoot()
		headRoot = parent[:]
		headState, err = vs.BeaconDB.State(ctx, parent)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		if headState == nil {
			if featureconfig.Get().NewStateMgmt {
				headState, err = vs.StateGen.StateByRoot(ctx, parent)
				if err != nil {
					return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to generate state: %v", err))
				}
			}
			if headState == nil {
				return nil, status.Error(codes.Internal, "Failed to lookup parent state from head.")
			}
		}
	}

	if helpers.CurrentEpoch(headState) < helpers.SlotToEpoch(req.Slot) {
		headState, err = state.ProcessSlots(ctx, headState, helpers.StartSlot(helpers.SlotToEpoch(req.Slot)))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not process slots up to %d: %v", req.Slot, err)
		}
	}

	targetEpoch := helpers.CurrentEpoch(headState)
	epochStartSlot := helpers.StartSlot(targetEpoch)
	targetRoot := make([]byte, 32)
	if epochStartSlot == headState.Slot() {
		targetRoot = headRoot[:]
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
		BeaconBlockRoot: headRoot[:],
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
	if _, err := bls.SignatureFromBytes(att.Signature); err != nil {
		return nil, status.Error(codes.InvalidArgument, "Incorrect attestation signature")
	}

	// If attestation committee subnets are enabled, we track the committee
	// index into a cache.
	if featureconfig.Get().EnableDynamicCommitteeSubnets {
		cache.CommitteeIDs.AddIDs([]uint64{att.Data.CommitteeIndex}, helpers.SlotToEpoch(att.Data.Slot))
	}

	root, err := ssz.HashTreeRoot(att.Data)
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

	// Broadcast the new attestation to the network.
	if err := vs.P2P.Broadcast(ctx, att); err != nil {
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

// waitToOneThird waits until one-third of the way through the slot
// or the head slot equals to the input slot.
func (vs *Server) waitToOneThird(ctx context.Context, slot uint64) {
	_, span := trace.StartSpan(ctx, "validator.waitToOneThird")
	defer span.End()

	// Don't need to wait if current slot is greater than requested slot.
	if slot < helpers.SlotsSince(vs.GenesisTime) {
		return
	}

	// Set time out to be at start slot time + one-third of slot duration.
	slotStartTime := slotutil.SlotStartTime(uint64(vs.GenesisTimeFetcher.GenesisTime().Unix()), slot)
	slotOneThirdTime := slotStartTime.Unix() + int64(params.BeaconConfig().SecondsPerSlot/3)
	waitDuration := slotOneThirdTime - roughtime.Now().Unix()
	timeOut := time.After(time.Duration(waitDuration) * time.Second)

	stateChannel := make(chan *feed.Event, 1)
	stateSub := vs.StateNotifier.StateFeed().Subscribe(stateChannel)
	defer stateSub.Unsubscribe()

	for {
		select {
		case event := <-stateChannel:
			// Node processed a block, check if the processed block is the same as input slot.
			if event.Type == statefeed.BlockProcessed {
				d := event.Data.(*statefeed.BlockProcessedData)
				if slot == d.Slot {
					return
				}
			}

		case <-timeOut:
			return
		}
	}
}
