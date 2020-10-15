package beaconv1

import (
	"context"
	"errors"

	ethpb_alpha "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
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
		v1Att, err := V1Alpha1AttestationToV1(att)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not migrate attestation: %v", err)
		}
		v1Atts[i] = v1Att
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
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	attSlashings := bs.SlashingsPool.PendingAttesterSlashings(ctx, headState, true /*noLimit*/)
	v1AttSlashings := make([]*ethpb.AttesterSlashing, len(attSlashings))
	for i, slashing := range attSlashings {
		v1Slashing, err := V1Alpha1AttSlashingToV1(slashing)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not migrate attester slashing: %v", err)
		}
		v1AttSlashings[i] = v1Slashing
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
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	slashings := bs.SlashingsPool.PendingProposerSlashings(ctx, headState, true /*noLimit*/)
	v1Slashings := make([]*ethpb.ProposerSlashing, len(slashings))
	for i, slashing := range slashings {
		v1Slashing, err := V1Alpha1ProposerSlashingToV1(slashing)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not migrate proposer slashing: %v", err)
		}
		v1Slashings[i] = v1Slashing
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
	headState, err := bs.HeadFetcher.HeadState(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	exits := bs.VoluntaryExitsPool.PendingExits(headState, headState.Slot(), true /*noLimit*/)
	v1Exits := make([]*ethpb.SignedVoluntaryExit, len(exits))
	for i, exit := range exits {
		v1Exit, err := V1Alpha1ExitToV1(exit)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Could not migrate exit: %v", err)
		}
		v1Exits[i] = v1Exit
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

func V1Alpha1AttestationToV1(v1alpha1Att *ethpb_alpha.Attestation) (*ethpb.Attestation, error) {
	marshaledAtt, err := v1alpha1Att.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal attestation: %v", err)
	}
	v1Att := &ethpb.Attestation{}
	if err := proto.Unmarshal(marshaledAtt, v1Att); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal attestation: %v", err)
	}
	return v1Att, nil
}

func V1Alpha1ExitToV1(v1alpha1Exit *ethpb_alpha.SignedVoluntaryExit) (*ethpb.SignedVoluntaryExit, error) {
	marshaledExit, err := v1alpha1Exit.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal voluntary exit: %v", err)
	}
	v1Exit := &ethpb.SignedVoluntaryExit{}
	if err := proto.Unmarshal(marshaledExit, v1Exit); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal voluntary exit: %v", err)
	}
	return v1Exit, nil
}

func V1Alpha1AttSlashingToV1(v1alpha1Slashing *ethpb_alpha.AttesterSlashing) (*ethpb.AttesterSlashing, error) {
	marshaledSlashing, err := v1alpha1Slashing.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal attester slashing: %v", err)
	}
	v1AttSlashing := &ethpb.AttesterSlashing{}
	if err := proto.Unmarshal(marshaledSlashing, v1AttSlashing); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal attester slashing: %v", err)
	}
	return v1AttSlashing, nil
}

func V1Alpha1ProposerSlashingToV1(v1alpha1Slashing *ethpb_alpha.ProposerSlashing) (*ethpb.ProposerSlashing, error) {
	marshaledSlashing, err := v1alpha1Slashing.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not marshal proposer slashing: %v", err)
	}
	v1Slashing := &ethpb.ProposerSlashing{}
	if err := proto.Unmarshal(marshaledSlashing, v1Slashing); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not unmarshal proposer slashing: %v", err)
	}
	return v1Slashing, nil
}
