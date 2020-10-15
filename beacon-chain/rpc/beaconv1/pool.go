package beaconv1

import (
	"context"
	"errors"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/migration"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListPoolAttestations retrieves attestations known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttestations(ctx context.Context, req *ethpb.AttestationsPoolRequest) (*ethpb.AttestationsPoolResponse, error) {
	atts := bs.AttestationsPool.AggregatedAttestations()
	filtered := make([]*ethpb_alpha.Attestation, 0, len(atts))
	for _, item := range atts {
		slotEqual := req.Slot != 0 && req.Slot == item.Data.Slot
		committeeEqual := req.CommitteeIndex != 0 && req.CommitteeIndex == item.Data.CommitteeIndex
		if slotEqual && committeeEqual {
			filtered = append(filtered, item)
		} else if slotEqual || committeeEqual {
			filtered = append(filtered, item)
		}
	}
	v1Atts := make([]*ethpb.Attestation, len(filtered))
	for i, att := range filtered {
		v1Atts[i] = migration.V1Alpha1AttestationToV1(att)
	}
	return &ethpb.AttestationsPoolResponse{
		Data: v1Atts,
	}, nil
}

// SubmitAttestation submits Attestation object to node. If attestation passes all validation
// constraints, node MUST publish attestation on appropriate subnet.
func (bs *Server) SubmitAttestation(ctx context.Context, req *ethpb.Attestation) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolAttesterSlashings retrieves attester slashings known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolAttesterSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.AttesterSlashingsPoolResponse, error) {
	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	attSlashings := bs.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /*noLimit*/)
	v1AttSlashings := make([]*ethpb.AttesterSlashing, len(attSlashings))
	for i, slashing := range attSlashings {
		v1AttSlashings[i] = migration.V1Alpha1AttSlashingToV1(slashing)
	}
	return &ethpb.AttesterSlashingsPoolResponse{
		Data: v1AttSlashings,
	}, nil
}

// SubmitAttesterSlashing submits AttesterSlashing object to node's pool and
// if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitAttesterSlashing(ctx context.Context, req *ethpb.AttesterSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolProposerSlashings retrieves proposer slashings known by the node
// but not necessarily incorporated into any block.
func (bs *Server) ListPoolProposerSlashings(ctx context.Context, req *ptypes.Empty) (*ethpb.ProposerSlashingPoolResponse, error) {
	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	slashings := bs.SlashingsPool.PendingProposerSlashings(ctx, headState, true /*noLimit*/)
	v1Slashings := make([]*ethpb.ProposerSlashing, len(slashings))
	for i, slashing := range slashings {
		v1Slashings[i] = migration.V1Alpha1ProposerSlashingToV1(slashing)
	}
	return &ethpb.ProposerSlashingPoolResponse{
		Data: v1Slashings,
	}, nil
}

// SubmitProposerSlashing submits AttesterSlashing object to node's pool and if
// passes validation node MUST broadcast it to network.
func (bs *Server) SubmitProposerSlashing(ctx context.Context, req *ethpb.ProposerSlashing) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}

// ListPoolVoluntaryExits retrieves voluntary exits known by the node but
// not necessarily incorporated into any block.
func (bs *Server) ListPoolVoluntaryExits(ctx context.Context, req *ptypes.Empty) (*ethpb.VoluntaryExitsPoolResponse, error) {
	headState, err := bs.ChainInfoFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	exits := bs.VoluntaryExitsPool.PendingExits(headState, headState.Slot(), true /*noLimit*/)
	v1Exits := make([]*ethpb.SignedVoluntaryExit, len(exits))
	for i, exit := range exits {
		v1Exits[i] = migration.V1Alpha1ExitToV1(exit)
	}
	return &ethpb.VoluntaryExitsPoolResponse{
		Data: v1Exits,
	}, nil
}

// SubmitVoluntaryExit submits SignedVoluntaryExit object to node's pool
// and if passes validation node MUST broadcast it to network.
func (bs *Server) SubmitVoluntaryExit(ctx context.Context, req *ethpb.SignedVoluntaryExit) (*ptypes.Empty, error) {
	return nil, errors.New("unimplemented")
}
