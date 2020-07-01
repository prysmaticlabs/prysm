package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// CommitBlock this function is part of slashing protection for block proposals it performs
// validation and db update.
func (s *Service) CommitBlock(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) bool {
	ps, err := s.slasherClient.IsSlashableBlock(ctx, blockHeader)
	if err != nil {
		log.Warnf("External slashing block protection returned an error: %v", err)
	}
	if ps != nil && ps.ProposerSlashing != nil {
		log.Warn("External slashing proposal protection found the block to be slashable")
		return false
	}
	return true
}

// VerifyBlock this function is part of slashing protection for block proposals it performs
// validation without db update.
func (s *Service) VerifyBlock(ctx context.Context, blockHeader *ethpb.BeaconBlockHeader) bool {
	slashable, err := s.slasherClient.IsSlashableBlockNoUpdate(ctx, blockHeader)
	if err != nil {
		log.Warnf("External slashing block protection returned an error: %v", err)
	}
	if slashable.Slashable {
		log.Warn("External slashing proposal protection found the block to be slashable")
	}
	return !slashable.Slashable
}

// VerifyAttestation implements the slashing protection for attestations without db update.
func (s *Service) VerifyAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	slashable, err := s.slasherClient.IsSlashableAttestationNoUpdate(ctx, attestation)
	if err != nil {
		log.Warnf("External slashing attestation protection returned an error: %v", err)
	}
	if slashable.Slashable {
		log.Warn("External slashing attestation protection found the attestation to be slashable")
	}
	return !slashable.Slashable
}

// CommitAttestation implements the slashing protection for attestations it performs
// validation and db update.
func (s *Service) CommitAttestation(ctx context.Context, attestation *ethpb.IndexedAttestation) bool {
	as, err := s.slasherClient.IsSlashableAttestation(ctx, attestation)
	if err != nil {
		log.Warnf("External slashing attestation protection returned an error: %v", err)
	}
	if as != nil && as.AttesterSlashing != nil {
		log.Warnf("External slashing attestation protection found the attestation to be slashable: %v", as)
		return false
	}
	return true
}
