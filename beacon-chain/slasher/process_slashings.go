package slasher

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

func (s *Service) processAttesterSlashings(ctx context.Context, slashings []*slashertypes.Slashing) {
	for _, attSlashing := range slashings {
		if err := s.verifyAttSignature(ctx, attSlashing.Attestation); err != nil {
			continue
		}
		if err := s.verifyAttSignature(ctx, attSlashing.PrevAttestation); err != nil {
			continue
		}
		// Log the slashing event.
		logSlashingEvent(attSlashing)

		// Submit to the slashing operations pool.
		// TODO: Implement.
	}
}

func (s *Service) verifyAttSignature(ctx context.Context, att *ethpb.IndexedAttestation) error {
	preState, err := s.serviceCfg.StateFetcher.AttestationPreState(ctx, att.Data.Target)
	if err != nil {
		return err
	}
	return blocks.VerifyIndexedAttestation(ctx, preState, att)
}
