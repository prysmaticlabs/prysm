package client

import (
	"context"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
)

var failedBlockSignLocalErr = "block rejected by local protection"

// slashableProposalCheck checks if a block proposal is slashable by comparing it with the
// block proposals history for the given public key in our DB. If it is not, we then update the history
// with new values and save it to the database.
func (v *validator) slashableProposalCheck(
	ctx context.Context,
	pubKey [fieldparams.BLSPubkeyLength]byte,
	signedBlock interfaces.ReadOnlySignedBeaconBlock,
	signingRoot [32]byte,
) error {
	if err := v.db.SlashableProposalCheck(ctx, pubKey, signedBlock, signingRoot, v.emitAccountMetrics, ValidatorProposeFailVec); err != nil {
		return errors.Wrapf(err, "could not check if block proposal is slashable for public key %#x", pubKey)
	}

	return nil
}
