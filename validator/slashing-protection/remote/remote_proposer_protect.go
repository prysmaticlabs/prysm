package remote

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection"
)

// IsSlashableBlock submits a block to a remote slasher instance to check whether it is
// slashable or not via a gRPC connection.
func (rp *Service) IsSlashableBlock(
	ctx context.Context, block *ethpb.SignedBeaconBlock, pubKey [48]byte, signingRoot [32]byte,
) (bool, error) {
	signedHeader, err := blockutil.SignedBeaconBlockHeaderFromBlock(block)
	if err != nil {
		return false, errors.Wrap(err, "could not extract signed header from block")
	}
	resp, err := rp.slasherClient.IsSlashableBlock(ctx, signedHeader)
	if err != nil {
		return false, parseSlasherError(err)
	}
	if resp != nil && resp.ProposerSlashing != nil {
		slashingprotection.RemoteSlashableProposalsTotal.Inc()
		return true, nil
	}
	return false, nil
}
