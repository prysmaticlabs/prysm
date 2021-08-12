package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// CheckBlockSafety this function is part of slashing protection for block proposals it performs
// validation without db update. To be used before the block is signed.
func (s *Service) CheckBlockSafety(ctx context.Context, blockHeader *ethpb.BeaconBlockHeader) bool {
	slashable, err := s.slasherClient.IsSlashableBlockNoUpdate(ctx, blockHeader) //nolint:staticcheck
	if err != nil {
		log.Errorf("External slashing block protection returned an error: %v", err)
		return false
	}
	//nolint:staticcheck
	if slashable != nil && slashable.Slashable {
		log.Warn("External slashing proposal protection found the block to be slashable")
	}
	//nolint:staticcheck
	return !slashable.Slashable
}

// CommitBlock this function is part of slashing protection for block proposals it performs
// validation and db update. To be used after the block is proposed.
func (s *Service) CommitBlock(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) (bool, error) {
	ps, err := s.slasherClient.IsSlashableBlock(ctx, blockHeader)
	if err != nil {
		log.Errorf("External slashing block protection returned an error: %v", err)
		return false, err
	}
	if ps != nil && len(ps.ProposerSlashings) != 0 {
		log.Warn("External slashing proposal protection found the block to be slashable")
		return false, nil
	}
	return true, nil
}

// CheckAttestationSafety implements the slashing protection for attestations without db update.
// To be used before signing.
func (s *Service) CheckAttestationSafety(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	slashable, err := s.slasherClient.IsSlashableAttestationNoUpdate(ctx, attestation) //nolint:staticcheck
	if err != nil {
		log.Errorf("External slashing attestation protection returned an error: %v", err)
		return false
	}
	//nolint:staticcheck
	if slashable.Slashable {
		log.Warn("External slashing attestation protection found the attestation to be slashable")
	}
	//nolint:staticcheck
	return !slashable.Slashable
}

// CommitAttestation implements the slashing protection for attestations it performs
// validation and db update. To be used after the attestation is proposed.
func (s *Service) CommitAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
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
