package slashingprotection

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// CheckAttestationSafety implements the slashing protection for attestations without db update.
// To be used before signing.
func (rp *RemoteProtector) IsSlashableAttestation(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	pubKey [48]byte,
	domain *ethpb.DomainResponse,
) (bool, error) {
	as, err := rp.slasherClient.IsSlashableAttestation(ctx, indexedAtt)
	if err != nil {
		log.Errorf("External slashing attestation protection returned an error: %v", err)
		return false, err
	}
	return as != nil && as.AttesterSlashing != nil, nil
}
