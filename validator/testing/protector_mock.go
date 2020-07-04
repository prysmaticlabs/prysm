package testing

import (
	"context"

	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// MockProtector mocks the protector.
type MockProtector struct {
	AllowAttestation        bool
	AllowBlock              bool
	VerifyAttestationCalled bool
	CommitAttestationCalled bool
	VerifyBlockCalled       bool
	CommitBlockCalled       bool
}

// CheckAttestationSafety returns bool with allow attestation value.
func (mp MockProtector) CheckAttestationSafety(ctx context.Context, attestation *eth.IndexedAttestation) bool {
	mp.VerifyAttestationCalled = true
	return mp.AllowAttestation
}

// CommitAttestation returns bool with allow attestation value.
func (mp MockProtector) CommitAttestation(ctx context.Context, attestation *eth.IndexedAttestation) bool {
	mp.CommitAttestationCalled = true
	return mp.AllowAttestation
}

// CheckBlockSafety returns bool with allow block value.
func (mp MockProtector) CheckBlockSafety(ctx context.Context, blockHeader *eth.BeaconBlockHeader) bool {
	mp.VerifyBlockCalled = true
	return mp.AllowBlock
}

// CommitBlock returns bool with allow block value.
func (mp MockProtector) CommitBlock(ctx context.Context, blockHeader *eth.SignedBeaconBlockHeader) bool {
	mp.CommitBlockCalled = true
	return mp.AllowBlock
}
