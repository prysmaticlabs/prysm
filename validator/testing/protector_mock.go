package testing

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashingprotection "github.com/prysmaticlabs/prysm/validator/slashing-protection"
)

var _ = slashingprotection.Protector(MockProtector{})

// MockProtector mocks the protector.
type MockProtector struct {
	SlashableAttestation         bool
	SlashableBlock               bool
	IsSlashableAttestationCalled bool
	IsSlashableBlockCalled       bool
}

// IsSlashableAttestation --
func (mp MockProtector) IsSlashableAttestation(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	pubKey [48]byte,
	domain *ethpb.DomainResponse,
) (bool, error) {
	return mp.SlashableAttestation, nil
}

// IsSlashableBlock --
func (mp MockProtector) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, domain *ethpb.DomainResponse,
) (bool, error) {
	return mp.SlashableBlock, nil
}

// Status returns nil.
func (mp MockProtector) Status() error {
	return nil
}

// Start returns.
func (mp MockProtector) Start() {}

// Stop returns nil.
func (mp MockProtector) Stop() error {
	return nil
}
