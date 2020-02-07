package slasher

import (
	"context"

	"google.golang.org/grpc"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

var _ = slashpb.SlasherClient(&fakeSlasher{})

type fakeSlasher struct {
	DoneCalled                   bool
	IsSlashableBlockCalled       bool
	IsSlashableAttestationCalled bool
	ProposerSlashingsCalled      bool
	AttesterSlashingsCalled      bool
}

func (fs *fakeSlasher) Done() {
	fs.DoneCalled = true
}
func (fs *fakeSlasher) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation, arg2 ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
	fs.IsSlashableAttestationCalled = true
	return &slashpb.AttesterSlashingResponse{}, nil
}
func (fs *fakeSlasher) IsSlashableBlock(ctx context.Context, in *slashpb.ProposerSlashingRequest, arg2 ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
	fs.IsSlashableBlockCalled = true
	return &slashpb.ProposerSlashingResponse{}, nil
}
func (fs *fakeSlasher) ProposerSlashings(ctx context.Context, in *slashpb.SlashingStatusRequest, arg2 ...grpc.CallOption) (*slashpb.ProposerSlashingResponse, error) {
	fs.ProposerSlashingsCalled = true
	return &slashpb.ProposerSlashingResponse{}, nil
}
func (fs *fakeSlasher) AttesterSlashings(ctx context.Context, in *slashpb.SlashingStatusRequest, arg2 ...grpc.CallOption) (*slashpb.AttesterSlashingResponse, error) {
	fs.AttesterSlashingsCalled = true
	return &slashpb.AttesterSlashingResponse{}, nil
}
