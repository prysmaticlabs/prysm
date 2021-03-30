package beaconv1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListPoolAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttestations(ctx context.Context, req *ethpb.AttestationsPoolRequest) (*ethpb.AttestationsPoolResponse, error) {
	return nil, errors.New("unimplemented")
}

// SubmitAttestation submits Attestation object to node. If attestation passes all validation
// constraints, node MUST publish attestation on appropriate subnet.
func (bs *Server) SubmitAttestations(ctx context.Context, req *ethpb.SubmitAttestationsRequest) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.SubmitAttestation")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	var validAttestations []*ethpb_alpha.Attestation
	for _, sourceAtt := range req.Data {
		att := migration.V1AttToV1Alpha1(sourceAtt)
		err = blocks.VerifyAttestationNoVerifySignature(ctx, headState, att)
		if err != nil {
			continue
		}
		err = blocks.VerifyAttestationSignature(ctx, headState, att)
		if err == nil {
			validAttestations = append(validAttestations, att)
		}
	}

	err = bs.AttestationsPool.SaveAggregatedAttestations(validAttestations)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not save attestations: %v", err)
	}
	broadcastFailed := false
	for _, att := range validAttestations {
		if err := bs.Broadcaster.Broadcast(ctx, att); err != nil {
			broadcastFailed = true
		}
	}
	if broadcastFailed {
		return nil, status.Errorf(
			codes.Internal,
			"Could not publish one or more attestations. Some attestations could be published successfully.")
	}
	return &ptypes.Empty{}, nil
}

// ListPoolAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttesterSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.AttesterSlashingsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.ListPoolAttesterSlashings")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	sourceSlashings := bs.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /* return unlimited slashings */)

	slashings := make([]*ethpb.AttesterSlashing, len(sourceSlashings))
	for i, s := range sourceSlashings {
		slashings[i] = migration.V1Alpha1AttSlashingToV1(s)
	}

	return &ethpb.AttesterSlashingsPoolResponse{
		Data: slashings,
	}, nil
}

// SubmitAttesterSlashing submits AttesterSlashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitAttesterSlashing(ctx context.Context, req *ethpb.AttesterSlashing) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.SubmitAttesterSlashing")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	alphaSlashing := migration.V1AttSlashingToV1Alpha1(req)
	err = blocks.VerifyAttesterSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Invalid attester slashing: %v", err)
	}

	err = bs.SlashingsPool.InsertAttesterSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert attester slashing into pool: %v", err)
	}
	if !featureconfig.Get().DisableBroadcastSlashings {
		if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not broadcast slashing object: %v", err)
		}
	}

	return &ptypes.Empty{}, nil
}

// ListPoolProposerSlashings retrieves proposer slashings known by the node
// but not necessarily incorporated into any block.
func (bs *Server) ListPoolProposerSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashingPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.ListPoolProposerSlashings")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	sourceSlashings := bs.SlashingsPool.PendingProposerSlashings(ctx, headState, true /* return unlimited slashings */)

	slashings := make([]*ethpb.ProposerSlashing, len(sourceSlashings))
	for i, s := range sourceSlashings {
		slashings[i] = migration.V1Alpha1ProposerSlashingToV1(s)
	}

	return &ethpb.ProposerSlashingPoolResponse{
		Data: slashings,
	}, nil
}

// SubmitProposerSlashing submits AttesterSlashing object to node's pool and if
// passes validation node MUST broadcast it to network.
func (bs *Server) SubmitProposerSlashing(ctx context.Context, req *ethpb.ProposerSlashing) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.SubmitProposerSlashing")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	alphaSlashing := migration.V1ProposerSlashingToV1Alpha1(req)
	err = blocks.VerifyProposerSlashing(headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Invalid proposer slashing: %v", err)
	}

	err = bs.SlashingsPool.InsertProposerSlashing(ctx, headState, alphaSlashing)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not insert proposer slashing into pool: %v", err)
	}
	if !featureconfig.Get().DisableBroadcastSlashings {
		if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
			return nil, status.Errorf(codes.Internal, "Could not broadcast slashing object: %v", err)
		}
	}

	return &ptypes.Empty{}, nil
}

// ListPoolVoluntaryExits retrieves voluntary exits known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolVoluntaryExits(ctx context.Context, req *ptypes.Empty) (*ethpb.VoluntaryExitsPoolResponse, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.ListPoolVoluntaryExits")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	sourceExits := bs.VoluntaryExitsPool.PendingExits(headState, headState.Slot(), true /* return unlimited exits */)

	exits := make([]*ethpb.SignedVoluntaryExit, len(sourceExits))
	for i, s := range sourceExits {
		exits[i] = migration.V1Alpha1ExitToV1(s)
	}

	return &ethpb.VoluntaryExitsPoolResponse{
		Data: exits,
	}, nil
}

// SubmitVoluntaryExit submits SignedVoluntaryExit object to node's pool
// and if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitVoluntaryExit(ctx context.Context, req *ethpb.SignedVoluntaryExit) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "beaconv1.SubmitVoluntaryExit")
	defer span.End()

	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}

	validator, err := headState.ValidatorAtIndexReadOnly(req.Exit.ValidatorIndex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get exiting validator: %v", err)
	}
	alphaExit := migration.V1ExitToV1Alpha1(req)
	err = blocks.VerifyExitAndSignature(validator, headState.Slot(), headState.Fork(), alphaExit, headState.GenesisValidatorRoot())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Invalid voluntary exit: %v", err)
	}

	bs.VoluntaryExitsPool.InsertVoluntaryExit(ctx, headState, alphaExit)
	if err := bs.Broadcaster.Broadcast(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast voluntary exit object: %v", err)
	}

	return &ptypes.Empty{}, nil
}
