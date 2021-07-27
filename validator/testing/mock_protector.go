package testing

import (
	"context"

	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// MockProtector mocks the protector.
type MockProtector struct {
	AllowAttestation        bool
	AllowBlock              bool
	VerifyAttestationCalled bool
	CommitAttestationCalled bool
	VerifyBlockCalled       bool
	CommitBlockCalled       bool
	StatusCalled            bool
}

// CheckAttestationSafety returns bool with allow attestation value.
func (mp MockProtector) CheckAttestationSafety(_ context.Context, _ *eth.IndexedAttestation) bool {
	mp.VerifyAttestationCalled = true
	return mp.AllowAttestation
}

// CommitAttestation returns bool with allow attestation value.
func (mp MockProtector) CommitAttestation(_ context.Context, _ *eth.IndexedAttestation) bool {
	mp.CommitAttestationCalled = true
	return mp.AllowAttestation
}

// CheckBlockSafety returns bool with allow block value.
func (mp MockProtector) CheckBlockSafety(_ context.Context, _ *eth.BeaconBlockHeader) bool {
	mp.VerifyBlockCalled = true
	return mp.AllowBlock
}

// CommitBlock returns bool with allow block value.
func (mp MockProtector) CommitBlock(_ context.Context, _ *eth.SignedBeaconBlockHeader) (bool, error) {
	mp.CommitBlockCalled = true
	return mp.AllowBlock, nil
}

// Status returns nil.
func (mp MockProtector) Status() error {
	mp.StatusCalled = true
	return nil
}
