package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epbs"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SubmitSignedExecutionPayloadEnvelope submits a signed execution payload envelope to the network.
func (vs *Server) SubmitSignedExecutionPayloadEnvelope(ctx context.Context, env *enginev1.SignedExecutionPayloadEnvelope) (*emptypb.Empty, error) {
	if err := vs.P2P.Broadcast(ctx, env); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to broadcast signed execution payload envelope: %v", err)
	}

	m, err := blocks.WrappedROExecutionPayloadEnvelope(env.Message)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to wrap execution payload envelope: %v", err)
	}

	if err := vs.ExecutionPayloadReceiver.ReceiveExecutionPayloadEnvelope(ctx, m, nil); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to receive execution payload envelope: %v", err)
	}

	return nil, nil
}

// SubmitSignedExecutionPayloadHeader submits a signed execution payload header to the beacon node.
func (vs *Server) SubmitSignedExecutionPayloadHeader(ctx context.Context, h *enginev1.SignedExecutionPayloadHeader) (*emptypb.Empty, error) {
	currentSlot := vs.TimeFetcher.CurrentSlot()
	if currentSlot != h.Message.Slot && currentSlot != h.Message.Slot-1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid slot: current slot %d, got %d", vs.TimeFetcher.CurrentSlot(), h.Message.Slot)
	}

	vs.signedExecutionPayloadHeader = h

	headState, err := vs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve head state: %v", err)
	}
	proposerIndex, err := helpers.BeaconProposerIndexAtSlot(ctx, headState, h.Message.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve proposer index: %v", err)
	}
	if proposerIndex != h.Message.BuilderIndex {
		if err := vs.P2P.Broadcast(ctx, h); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to broadcast signed execution payload header: %v", err)
		}
	}

	return nil, nil
}

// computePostPayloadStateRoot computes the state root after an execution
// payload envelope has been processed through a state transition and
// returns it to the validator client.
func (vs *Server) computePostPayloadStateRoot(ctx context.Context, envelope interfaces.ROExecutionPayloadEnvelope) ([]byte, error) {
	beaconState, err := vs.StateGen.StateByRoot(ctx, envelope.BeaconBlockRoot())
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve beacon state")
	}
	beaconState = beaconState.Copy()
	err = epbs.ProcessPayloadStateTransition(
		ctx,
		beaconState,
		envelope,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate post payload state root at slot %d", beaconState.Slot())
	}

	root, err := beaconState.HashTreeRoot(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not calculate post payload state root at slot %d", beaconState.Slot())
	}
	log.WithField("beaconStateRoot", fmt.Sprintf("%#x", root)).Debugf("Computed state root at execution stage")
	return root[:], nil
}

// GetLocalHeader returns the local header for a given slot and proposer index.
func (vs *Server) GetLocalHeader(ctx context.Context, req *eth.HeaderRequest) (*enginev1.ExecutionPayloadHeaderEPBS, error) {
	ctx, span := trace.StartSpan(ctx, "ProposerServer.GetLocalHeader")
	defer span.End()

	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.FailedPrecondition, "Syncing to latest head, not ready to respond")
	}

	if err := vs.optimisticStatus(ctx); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Validator is not ready to propose: %v", err)
	}

	slot := req.Slot
	epoch := slots.ToEpoch(slot)
	if params.BeaconConfig().EPBSForkEpoch > epoch {
		return nil, status.Errorf(codes.FailedPrecondition, "EPBS fork has not occurred yet")
	}

	st, parentRoot, err := vs.getParentState(ctx, slot)
	if err != nil {
		return nil, err
	}

	proposerIndex := req.ProposerIndex
	localPayload, err := vs.getLocalPayloadFromEngine(ctx, st, parentRoot, slot, proposerIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get local payload: %v", err)
	}

	kzgRoot, err := ssz.KzgCommitmentsRoot(localPayload.BlobsBundle.KzgCommitments)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get kzg commitments root: %v", err)
	}

	return &enginev1.ExecutionPayloadHeaderEPBS{
		ParentBlockHash:        localPayload.ExecutionData.ParentHash(),
		ParentBlockRoot:        parentRoot[:],
		BlockHash:              localPayload.ExecutionData.BlockHash(),
		GasLimit:               localPayload.ExecutionData.GasLimit(),
		BuilderIndex:           proposerIndex,
		Slot:                   slot,
		Value:                  0,
		BlobKzgCommitmentsRoot: kzgRoot[:],
	}, nil
}
