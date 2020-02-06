package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1_gateway"
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
func (fs *fakeSlasher) IsSlashableAttestation(ctx context.Context, in *ethpb.IndexedAttestation) (*slashpb.AttesterSlashingResponse, error) {
	fs.IsSlashableAttestationCalled = true
	return nil, nil
}
func (fs *fakeSlasher) IsSlashableBlock(ctx context.Context, in *slashpb.ProposerSlashingRequest) (*slashpb.ProposerSlashingResponse, error) {
	fs.IsSlashableBlockCalled = true
	return nil, nil
}
func (fs *fakeSlasher) ProposerSlashings(ctx context.Context, in *slashpb.SlashingStatusRequest) (*slashpb.ProposerSlashingResponse, error) {
	fs.ProposerSlashingsCalled = true
	return nil, nil
}
func (fs *fakeSlasher) AttesterSlashings(ctx context.Context, in *slashpb.SlashingStatusRequest) (*slashpb.AttesterSlashingResponse, error) {
	fs.AttesterSlashingsCalled = true
	return nil, nil
}
