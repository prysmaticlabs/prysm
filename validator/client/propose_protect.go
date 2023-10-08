package client

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/sirupsen/logrus"
)

var failedBlockSignLocalErr = "attempted to sign a double proposal, block rejected by local protection"

func (v *validator) slashableProposalCheck(
	ctx context.Context, pubKey [fieldparams.BLSPubkeyLength]byte, signedBlock interfaces.ReadOnlySignedBeaconBlock, signingRoot [32]byte,
) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])

	blk := signedBlock.Block()
	prevSigningRoot, proposalAtSlotExists, err := v.db.ProposalHistoryForSlot(ctx, pubKey, blk.Slot())
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.Wrap(err, "failed to get proposal history")
	}

	lowestSignedProposalSlot, lowestProposalExists, err := v.db.LowestSignedProposal(ctx, pubKey)
	if err != nil {
		return err
	}

	// If a proposal exists in our history for the slot, we check the following:
	// If the signing root is empty (zero hash), then we consider it slashable. If signing root is not empty,
	// we check if it is different than the incoming block's signing root. If that is the case,
	// we consider that proposal slashable.
	signingRootIsDifferent := prevSigningRoot == params.BeaconConfig().ZeroHash || prevSigningRoot != signingRoot
	if proposalAtSlotExists && signingRootIsDifferent {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedBlockSignLocalErr)
	}

	// Based on EIP3076, validator should refuse to sign any proposal with slot less
	// than or equal to the minimum signed proposal present in the DB for that public key.
	// In the case the slot of the incoming block is equal to the minimum signed proposal, we
	// then also check the signing root is different.
	if lowestProposalExists && signingRootIsDifferent && lowestSignedProposalSlot >= blk.Slot() {
		return fmt.Errorf(
			"could not sign block with slot <= lowest signed slot in db, lowest signed slot: %d >= block slot: %d",
			lowestSignedProposalSlot,
			blk.Slot(),
		)
	}

	if err := v.db.SaveProposalHistoryForSlot(ctx, pubKey, blk.Slot(), signingRoot[:]); err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.Wrap(err, "failed to save updated proposal history")
	}
	return nil
}

func blockLogFields(pubKey [fieldparams.BLSPubkeyLength]byte, blk interfaces.ReadOnlyBeaconBlock, sig []byte) logrus.Fields {
	fields := logrus.Fields{
		"proposerPublicKey": fmt.Sprintf("%#x", pubKey),
		"proposerIndex":     blk.ProposerIndex(),
		"blockSlot":         blk.Slot(),
	}
	if sig != nil {
		fields["signature"] = fmt.Sprintf("%#x", sig)
	}
	return fields
}
