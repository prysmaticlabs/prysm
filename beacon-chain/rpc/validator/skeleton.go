package validator

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//
// These skeletons should be replaced with implementations.
//

// GetDuties is a skeleton for the GetDuties GRPC call.
func (vs *Server) GetDuties(context.Context, *ethpb.DutiesRequest) (*ethpb.DutiesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

// GetBlock is a skeleton for the GetBlock GRPC call.
func (vs *Server) GetBlock(context.Context, *ethpb.BlockRequest) (*ethpb.BeaconBlock, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

// ProposeBlock is a skeleton for the ProposeBlock GRPC call.
func (vs *Server) ProposeBlock(context.Context, *ethpb.BlockRequest) (*ptypes.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

// GetAttestationData is a skeleton for the GetAttestationData GRPC call.
func (vs *Server) GetAttestationData(context.Context, *ethpb.AttestationDataRequest) (*ethpb.Attestation, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}

// ProposeAttestation is a skeleton for the ProposeAttestation GRPC call.
func (vs *Server) ProposeAttestation(context.Context, *ethpb.Attestation) (*ptypes.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented")
}
