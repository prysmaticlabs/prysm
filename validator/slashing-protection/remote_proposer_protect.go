package slashingprotection

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
)

// CheckBlockSafety this function is part of slashing protection for block proposals it performs
// validation without db update. To be used before the block is signed.
func (rp *RemoteProtector) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, domain *ethpb.DomainResponse,
) (bool, error) {
	signedHeader, err := blockutil.SignedBeaconBlockHeaderFromBlock(block)
	if err != nil {
		return false, errors.Wrap(err, "could not extract signed header from block")
	}
	resp, err := rp.slasherClient.IsSlashableBlock(ctx, signedHeader)
	if err != nil {
		return false, errors.Wrap(err, "remote slashing block protection returned an error")
	}
	if resp != nil && resp.ProposerSlashing != nil {
		remoteSlashableProposalsTotal.Inc()
		return true, nil
	}
	return false, nil
}
