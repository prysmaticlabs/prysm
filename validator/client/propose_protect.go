package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var failedPreBlockSignLocalErr = "attempted to sign a double proposal, block rejected by local protection"
var failedPreBlockSignExternalErr = "attempted a double proposal, block rejected by remote slashing protection"
var failedPostBlockSignErr = "made a double proposal, considered slashable by remote slashing protection"

func (v *validator) preBlockSignValidations(ctx context.Context, pubKey [48]byte, block *ethpb.BeaconBlock) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	signingRoot, err := v.db.ProposalHistoryForSlot(ctx, pubKey[:], block.Slot)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.Wrap(err, "failed to get proposal history")
	}
	// If the bit for the current slot is marked, do not propose.
	if !bytes.Equal(signingRoot, params.BeaconConfig().ZeroHash[:]) {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.New(failedPreBlockSignLocalErr)
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		blockHdr, err := blockutil.BeaconBlockHeaderFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to get block header from block")
		}
		if !v.protector.CheckBlockSafety(ctx, blockHdr) {
			if v.emitAccountMetrics {
				ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPreBlockSignExternalErr)
		}
	}

	return nil
}

func (v *validator) postBlockSignUpdate(ctx context.Context, pubKey [48]byte, block *ethpb.SignedBeaconBlock, domain *ethpb.DomainResponse) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	if featureconfig.Get().SlasherProtection && v.protector != nil {
		sbh, err := blockutil.SignedBeaconBlockHeaderFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to get block header from block")
		}
		valid, err := v.protector.CommitBlock(ctx, sbh)
		if err != nil {
			return err
		}
		if !valid {
			if v.emitAccountMetrics {
				ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf(failedPostBlockSignErr)
		}
	}
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, domain.SignatureDomain)
	if err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.Wrap(err, "failed to compute signing root for block")
	}
	if err := v.db.SaveProposalHistoryForSlot(ctx, pubKey[:], block.Block.Slot, signingRoot[:]); err != nil {
		if v.emitAccountMetrics {
			ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
		}
		return errors.Wrap(err, "failed to save updated proposal history")
	}
	return nil
}
