package beacon

import (
	"context"
	"time"

	"github.com/prysmaticlabs/prysm/v4/api/grpc"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	ethpbv1 "github.com/prysmaticlabs/prysm/v4/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/v4/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const broadcastBLSChangesRateLimit = 128

// ListPoolAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttesterSlashings(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.AttesterSlashingsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListPoolAttesterSlashings")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	sourceSlashings := bs.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /* return unlimited slashings */)

	slashings := make([]*ethpbv1.AttesterSlashing, len(sourceSlashings))
	for i, s := range sourceSlashings {
		slashings[i] = migration.V1Alpha1AttSlashingToV1(s)
	}

	return &ethpbv1.AttesterSlashingsPoolResponse{
		Data: slashings,
	}, nil
}

// SubmitAttesterSlashing submits AttesterSlashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitAttesterSlashing(ctx context.Context, req *ethpbv1.AttesterSlashing) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitAttesterSlashing")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, req.Attestation_1.Data.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots: %v", err)
	}

	alphaSlashing := migration.V1AttSlashingToV1Alpha1(req)
	err = blocks.VerifyAttesterSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid attester slashing: %v", err)
	}

	err = bs.SlashingsPool.InsertAttesterSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert attester slashing into pool: %v", err)
	}
	if !features.Get().DisableBroadcastSlashings {
		if err := bs.Broadcaster.Broadcast(ctx, alphaSlashing); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not broadcast slashing object: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}

// ListPoolProposerSlashings retrieves proposer slashings known by the node
// but not necessarily incorporated into any block.
func (bs *Server) ListPoolProposerSlashings(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.ProposerSlashingPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListPoolProposerSlashings")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	sourceSlashings := bs.SlashingsPool.PendingProposerSlashings(ctx, headState, true /* return unlimited slashings */)

	slashings := make([]*ethpbv1.ProposerSlashing, len(sourceSlashings))
	for i, s := range sourceSlashings {
		slashings[i] = migration.V1Alpha1ProposerSlashingToV1(s)
	}

	return &ethpbv1.ProposerSlashingPoolResponse{
		Data: slashings,
	}, nil
}

// SubmitProposerSlashing submits AttesterSlashing object to node's pool and if
// passes validation node MUST broadcast it to network.
func (bs *Server) SubmitProposerSlashing(ctx context.Context, req *ethpbv1.ProposerSlashing) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitProposerSlashing")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, req.SignedHeader_1.Message.Slot)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots: %v", err)
	}

	alphaSlashing := migration.V1ProposerSlashingToV1Alpha1(req)
	err = blocks.VerifyProposerSlashing(headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid proposer slashing: %v", err)
	}

	err = bs.SlashingsPool.InsertProposerSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert proposer slashing into pool: %v", err)
	}
	if !features.Get().DisableBroadcastSlashings {
		if err := bs.Broadcaster.Broadcast(ctx, alphaSlashing); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not broadcast slashing object: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}

// SubmitSignedBLSToExecutionChanges submits said object to the node's pool
// if it passes validation the node must broadcast it to the network.
func (bs *Server) SubmitSignedBLSToExecutionChanges(ctx context.Context, req *ethpbv2.SubmitBLSToExecutionChangesRequest) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitSignedBLSToExecutionChanges")
	defer span.End()
	st, err := bs.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	var failures []*helpers.SingleIndexedVerificationFailure
	var toBroadcast []*ethpbalpha.SignedBLSToExecutionChange

	for i, change := range req.GetChanges() {
		alphaChange := migration.V2SignedBLSToExecutionChangeToV1Alpha1(change)
		_, err = blocks.ValidateBLSToExecutionChange(st, alphaChange)
		if err != nil {
			failures = append(failures, &helpers.SingleIndexedVerificationFailure{
				Index:   i,
				Message: "Could not validate SignedBLSToExecutionChange: " + err.Error(),
			})
			continue
		}
		if err := blocks.VerifyBLSChangeSignature(st, change); err != nil {
			failures = append(failures, &helpers.SingleIndexedVerificationFailure{
				Index:   i,
				Message: "Could not validate signature: " + err.Error(),
			})
			continue
		}
		bs.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: operation.BLSToExecutionChangeReceived,
			Data: &operation.BLSToExecutionChangeReceivedData{
				Change: alphaChange,
			},
		})
		bs.BLSChangesPool.InsertBLSToExecChange(alphaChange)
		if st.Version() >= version.Capella {
			toBroadcast = append(toBroadcast, alphaChange)
		}
	}
	go bs.broadcastBLSChanges(ctx, toBroadcast)
	if len(failures) > 0 {
		failuresContainer := &helpers.IndexedVerificationFailure{Failures: failures}
		err := grpc.AppendCustomErrorHeader(ctx, failuresContainer)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"One or more BLSToExecutionChange failed validation. Could not prepare BLSToExecutionChange failure information: %v",
				err,
			)
		}
		return nil, status.Errorf(codes.InvalidArgument, "One or more BLSToExecutionChange failed validation")
	}
	return &emptypb.Empty{}, nil
}

// broadcastBLSBatch broadcasts the first `broadcastBLSChangesRateLimit` messages from the slice pointed to by ptr.
// It validates the messages again because they could have been invalidated by being included in blocks since the last validation.
// It removes the messages from the slice and modifies it in place.
func (bs *Server) broadcastBLSBatch(ctx context.Context, ptr *[]*ethpbalpha.SignedBLSToExecutionChange) {
	limit := broadcastBLSChangesRateLimit
	if len(*ptr) < broadcastBLSChangesRateLimit {
		limit = len(*ptr)
	}
	st, err := bs.ChainInfoFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		log.WithError(err).Error("could not get head state")
		return
	}
	for _, ch := range (*ptr)[:limit] {
		if ch != nil {
			_, err := blocks.ValidateBLSToExecutionChange(st, ch)
			if err != nil {
				log.WithError(err).Error("could not validate BLS to execution change")
				continue
			}
			if err := bs.Broadcaster.Broadcast(ctx, ch); err != nil {
				log.WithError(err).Error("could not broadcast BLS to execution changes.")
			}
		}
	}
	*ptr = (*ptr)[limit:]
}

func (bs *Server) broadcastBLSChanges(ctx context.Context, changes []*ethpbalpha.SignedBLSToExecutionChange) {
	bs.broadcastBLSBatch(ctx, &changes)
	if len(changes) == 0 {
		return
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bs.broadcastBLSBatch(ctx, &changes)
			if len(changes) == 0 {
				return
			}
		}
	}
}

// ListBLSToExecutionChanges retrieves BLS to execution changes known by the node but not necessarily incorporated into any block
func (bs *Server) ListBLSToExecutionChanges(ctx context.Context, _ *emptypb.Empty) (*ethpbv2.BLSToExecutionChangesPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListBLSToExecutionChanges")
	defer span.End()

	sourceChanges, err := bs.BLSChangesPool.PendingBLSToExecChanges()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get BLS to execution changes: %v", err)
	}

	changes := make([]*ethpbv2.SignedBLSToExecutionChange, len(sourceChanges))
	for i, ch := range sourceChanges {
		changes[i] = migration.V1Alpha1SignedBLSToExecChangeToV2(ch)
	}

	return &ethpbv2.BLSToExecutionChangesPoolResponse{
		Data: changes,
	}, nil
}
