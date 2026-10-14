package validator

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/decoupled"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AvailableAttestationData serves the head captured at Δ. Other slots and unjoined nodes get Unavailable.
func (vs *Server) AvailableAttestationData(_ context.Context, req *ethpb.DecoupledAttestationDataRequest) (*ethpb.AvailableAttestationData, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	root, ok := vs.ForkchoiceFetcher.DecoupledVote(req.Slot)
	if !ok {
		return nil, status.Errorf(codes.Unavailable, "no vote captured for slot %d", req.Slot)
	}
	return &ethpb.AvailableAttestationData{Slot: req.Slot, BeaconBlockRoot: root[:]}, nil
}

// SubmitAvailableAttestation receives a signed AC vote and broadcasts it.
func (vs *Server) SubmitAvailableAttestation(ctx context.Context, msg *ethpb.AvailableAttestationMessage) (*emptypb.Empty, error) {
	if msg == nil || msg.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "available attestation is nil")
	}
	st, err := vs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if err := decoupled.VerifyAvailableAttestation(st, msg); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid available attestation: %v", err)
	}
	// Receive locally before broadcasting, a slow broadcast must not delay the stamp.
	if err := vs.DecoupledReceiver.ReceiveAvailableAttestation(ctx, msg); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not process available attestation: %v", err)
	}
	if err := vs.P2P.Broadcast(ctx, msg); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast available attestation: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// FinalityAttestationData serves the source and head captured at the action time.
func (vs *Server) FinalityAttestationData(_ context.Context, req *ethpb.DecoupledAttestationDataRequest) (*ethpb.FinalityAttestationData, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if vs.SyncChecker.Syncing() {
		return nil, status.Error(codes.Unavailable, "Syncing to latest head, not ready to respond")
	}
	data, ok := vs.ForkchoiceFetcher.DecoupledFinalityData(req.Slot)
	if !ok {
		return nil, status.Errorf(codes.Unavailable, "no finality source captured for slot %d", req.Slot)
	}
	return data, nil
}

// SubmitFinalityAttestation pools a signed FG vote and broadcasts it.
func (vs *Server) SubmitFinalityAttestation(ctx context.Context, msg *ethpb.FinalityAttestationMessage) (*emptypb.Empty, error) {
	if msg == nil || msg.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "finality attestation is nil")
	}
	st, err := vs.HeadFetcher.HeadStateReadOnly(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head state: %v", err)
	}
	if err := decoupled.VerifyFinalityAttestation(st, msg); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid finality attestation: %v", err)
	}
	if err := vs.DecoupledFGPool.InsertSingle(msg.ValidatorIndex, msg.Data, msg.Signature); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not pool finality attestation: %v", err)
	}
	if err := vs.P2P.Broadcast(ctx, msg); err != nil {
		return nil, status.Errorf(codes.Internal, "Could not broadcast finality attestation: %v", err)
	}
	return &emptypb.Empty{}, nil
}
