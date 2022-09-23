package beacon

import (
	"context"

	"github.com/prysmaticlabs/prysm/v3/api/grpc"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/operation"
	corehelpers "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/config/features"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	ethpbv1 "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	ethpbalpha "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ListPoolAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block. Allows filtering by committee index or slot.
func (bs *Server) ListPoolAttestations(ctx context.Context, req *ethpbv1.AttestationsPoolRequest) (*ethpbv1.AttestationsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListPoolAttestations")
	defer span.End()

	attestations := bs.AttestationsPool.AggregatedAttestations()
	unaggAtts, err := bs.AttestationsPool.UnaggregatedAttestations()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get unaggregated attestations: %v", err)
	}
	attestations = append(attestations, unaggAtts...)
	isEmptyReq := req.Slot == nil && req.CommitteeIndex == nil
	if isEmptyReq {
		allAtts := make([]*ethpbv1.Attestation, len(attestations))
		for i, att := range attestations {
			allAtts[i] = migration.V1Alpha1AttestationToV1(att)
		}
		return &ethpbv1.AttestationsPoolResponse{Data: allAtts}, nil
	}

	filteredAtts := make([]*ethpbv1.Attestation, 0, len(attestations))
	for _, att := range attestations {
		bothDefined := req.Slot != nil && req.CommitteeIndex != nil
		committeeIndexMatch := req.CommitteeIndex != nil && att.Data.CommitteeIndex == *req.CommitteeIndex
		slotMatch := req.Slot != nil && att.Data.Slot == *req.Slot

		if bothDefined && committeeIndexMatch && slotMatch {
			filteredAtts = append(filteredAtts, migration.V1Alpha1AttestationToV1(att))
		} else if !bothDefined && (committeeIndexMatch || slotMatch) {
			filteredAtts = append(filteredAtts, migration.V1Alpha1AttestationToV1(att))
		}
	}
	return &ethpbv1.AttestationsPoolResponse{Data: filteredAtts}, nil
}

// SubmitAttestations submits Attestation object to node. If attestation passes all validation
// constraints, node MUST publish attestation on appropriate subnet.
func (bs *Server) SubmitAttestations(ctx context.Context, req *ethpbv1.SubmitAttestationsRequest) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitAttestation")
	defer span.End()

	var validAttestations []*ethpbalpha.Attestation
	var attFailures []*helpers.SingleIndexedVerificationFailure
	for i, sourceAtt := range req.Data {
		att := migration.V1AttToV1Alpha1(sourceAtt)
		if _, err := bls.SignatureFromBytes(att.Signature); err != nil {
			attFailures = append(attFailures, &helpers.SingleIndexedVerificationFailure{
				Index:   i,
				Message: "Incorrect attestation signature: " + err.Error(),
			})
			continue
		}

		// Broadcast the unaggregated attestation on a feed to notify other services in the beacon node
		// of a received unaggregated attestation.
		bs.OperationNotifier.OperationFeed().Send(&feed.Event{
			Type: operation.UnaggregatedAttReceived,
			Data: &operation.UnAggregatedAttReceivedData{
				Attestation: att,
			},
		})

		validAttestations = append(validAttestations, att)

		go func() {
			ctx = trace.NewContext(context.Background(), trace.FromContext(ctx))
			attCopy := ethpbalpha.CopyAttestation(att)
			if err := bs.AttestationsPool.SaveUnaggregatedAttestation(attCopy); err != nil {
				log.WithError(err).Error("Could not handle attestation in operations service")
				return
			}
		}()
	}

	broadcastFailed := false
	for _, att := range validAttestations {
		// Determine subnet to broadcast attestation to
		wantedEpoch := slots.ToEpoch(att.Data.Slot)
		vals, err := bs.HeadFetcher.HeadValidatorsIndices(ctx, wantedEpoch)
		if err != nil {
			return nil, err
		}
		subnet := corehelpers.ComputeSubnetFromCommitteeAndSlot(uint64(len(vals)), att.Data.CommitteeIndex, att.Data.Slot)

		if err := bs.Broadcaster.BroadcastAttestation(ctx, subnet, att); err != nil {
			broadcastFailed = true
		}
	}
	if broadcastFailed {
		return nil, status.Errorf(
			codes.Internal,
			"Could not publish one or more attestations. Some attestations could be published successfully.")
	}

	if len(attFailures) > 0 {
		failuresContainer := &helpers.IndexedVerificationFailure{Failures: attFailures}
		err := grpc.AppendCustomErrorHeader(ctx, failuresContainer)
		if err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"One or more attestations failed validation. Could not prepare attestation failure information: %v",
				err,
			)
		}
		return nil, status.Errorf(codes.InvalidArgument, "One or more attestations failed validation")
	}

	return &emptypb.Empty{}, nil
}

// ListPoolAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttesterSlashings(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.AttesterSlashingsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListPoolAttesterSlashings")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
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
		if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
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

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
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
		if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not broadcast slashing object: %v", err)
		}
	}

	return &emptypb.Empty{}, nil
}

// ListPoolVoluntaryExits retrieves voluntary exits known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolVoluntaryExits(ctx context.Context, _ *emptypb.Empty) (*ethpbv1.VoluntaryExitsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.ListPoolVoluntaryExits")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	sourceExits := bs.VoluntaryExitsPool.PendingExits(headState, headState.Slot(), true /* return unlimited exits */)

	exits := make([]*ethpbv1.SignedVoluntaryExit, len(sourceExits))
	for i, s := range sourceExits {
		exits[i] = migration.V1Alpha1ExitToV1(s)
	}

	return &ethpbv1.VoluntaryExitsPoolResponse{
		Data: exits,
	}, nil
}

// SubmitVoluntaryExit submits SignedVoluntaryExit object to node's pool
// and if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitVoluntaryExit(ctx context.Context, req *ethpbv1.SignedVoluntaryExit) (*emptypb.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beacon.SubmitVoluntaryExit")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	s, err := slots.EpochStart(req.Message.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get epoch from message: %v", err)
	}
	headState, err = transition.ProcessSlotsIfPossible(ctx, headState, s)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process slots: %v", err)
	}

	validator, err := headState.ValidatorAtIndexReadOnly(req.Message.ValidatorIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get exiting validator: %v", err)
	}
	alphaExit := migration.V1ExitToV1Alpha1(req)
	err = blocks.VerifyExitAndSignature(validator, headState.Slot(), headState.Fork(), alphaExit, headState.GenesisValidatorsRoot())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid voluntary exit: %v", err)
	}

	bs.VoluntaryExitsPool.InsertVoluntaryExit(ctx, headState, alphaExit)
	if err := bs.Broadcaster.Broadcast(ctx, alphaExit); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast voluntary exit object: %v", err)
	}

	return &emptypb.Empty{}, nil
}
