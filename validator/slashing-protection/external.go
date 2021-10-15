package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// CheckBlockSafety for blocks before submitting them to the node.
func (s *Service) CheckBlockSafety(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) bool {
	ps, err := s.slasherClient.IsSlashableBlock(ctx, blockHeader)
	if err != nil {
		log.Errorf("External slashing block protection returned an error: %v", err)
		return false
	}
	if ps != nil && len(ps.ProposerSlashings) != 0 {
		log.Warn("External slashing proposal protection found the block to be slashable")
		return false
	}
	return true
}

// CheckAttestationSafety for attestations before submitting them to the node.
func (s *Service) CheckAttestationSafety(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	as, err := s.slasherClient.IsSlashableAttestation(ctx, attestation)
	if err != nil {
		log.Errorf("External slashing attestation protection returned an error: %v", err)
		return false
	}
	if as != nil && len(as.AttesterSlashings) != 0 {
		log.Warnf("External slashing attestation protection found the attestation to be slashable: %v", as)
		return false
	}
	return true
}
