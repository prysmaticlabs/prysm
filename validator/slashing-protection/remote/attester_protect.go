package remote

import (
	"context"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection"
)

// IsSlashableAttestation submits an attestation to a remote slasher instance to check
// whether it is slashable or not via a gRPC connection.
func (rp *Service) IsSlashableAttestation(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	pubKey [48]byte,
	signingRoot [32]byte,
) (bool, error) {
	as, err := rp.slasherClient.IsSlashableAttestation(ctx, indexedAtt)
	if err != nil {
		return false, parseSlasherError(err)
	}
	if as != nil && as.AttesterSlashing != nil {
		slashingprotection.RemoteSlashableAttestationsTotal.Inc()
		return true, nil
	}
	return false, nil
}
