package client

import (
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
	epoch := helpers.SlotToEpoch(block.Slot)
	if featureconfig.Get().ProtectProposer {
		slotBits, err := v.db.ProposalHistoryForEpoch(ctx, pubKey[:], epoch)
		if err != nil {
			if v.emitAccountMetrics {
				ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to get proposal history")
		}

		// If the bit for the current slot is marked, do not propose.
		if slotBits.BitAt(block.Slot % params.BeaconConfig().SlotsPerEpoch) {
			if v.emitAccountMetrics {
				ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.New(failedPreBlockSignLocalErr)
		}
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

func (v *validator) postBlockSignUpdate(ctx context.Context, pubKey [48]byte, block *ethpb.SignedBeaconBlock) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	epoch := helpers.SlotToEpoch(block.Block.Slot)
	if featureconfig.Get().SlasherProtection && v.protector != nil {
		sbh, err := blockutil.SignedBeaconBlockHeaderFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to get block header from block")
		}
		if !v.protector.CommitBlock(ctx, sbh) {
			if v.emitAccountMetrics {
				ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf(failedPostBlockSignErr)
		}
	}

	if featureconfig.Get().ProtectProposer {
		slotBits, err := v.db.ProposalHistoryForEpoch(ctx, pubKey[:], epoch)
		if err != nil {
			if v.emitAccountMetrics {
				ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to get proposal history")
		}
		slotBits.SetBitAt(block.Block.Slot%params.BeaconConfig().SlotsPerEpoch, true)
		if err := v.db.SaveProposalHistoryForEpoch(ctx, pubKey[:], epoch, slotBits); err != nil {
			if v.emitAccountMetrics {
				ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to save updated proposal history")
		}
	}
	return nil
}
