package polling

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/blockutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/validator/client/metrics"
)

func (v *validator) preBlockSignValidations(ctx context.Context, pubKey [48]byte, block *ethpb.BeaconBlock) error {
	fmtKey := fmt.Sprintf("%#x", pubKey[:])
	epoch := helpers.SlotToEpoch(block.Slot)
	if featureconfig.Get().ProtectProposer {
		slotBits, err := v.db.ProposalHistoryForEpoch(ctx, pubKey[:], epoch)
		if err != nil {
			if v.emitAccountMetrics {
				metrics.ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to get proposal history")
		}

		// If the bit for the current slot is marked, do not propose.
		if slotBits.BitAt(block.Slot % params.BeaconConfig().SlotsPerEpoch) {
			if v.emitAccountMetrics {
				metrics.ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf("attempted to sign a double proposal, block rejected by local protection slot=%d", block.Slot)
		}
	}

	if featureconfig.Get().SlasherProtection && v.protector != nil {
		blockHdr, err := blockutil.BeaconBlockHeaderFromBlock(block)
		if err != nil {
			return errors.Wrap(err, "failed to get block header from block")
		}
		if !v.protector.CheckBlockSafety(ctx, blockHdr) {
			if v.emitAccountMetrics {
				metrics.ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf("attempted a double proposal, block rejected by remote slashing protection slot=%d", block.Slot)
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
				metrics.ValidatorProposeFailVecSlasher.WithLabelValues(fmtKey).Inc()
			}
			return fmt.Errorf("made a double proposal, considered slashable by remote slashing protection slot=%d", block.Block.Slot)
		}
	}

	if featureconfig.Get().ProtectProposer {
		slotBits, err := v.db.ProposalHistoryForEpoch(ctx, pubKey[:], epoch)
		if err != nil {
			if v.emitAccountMetrics {
				metrics.ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to get proposal history")
		}
		slotBits.SetBitAt(block.Block.Slot%params.BeaconConfig().SlotsPerEpoch, true)
		if err := v.db.SaveProposalHistoryForEpoch(ctx, pubKey[:], epoch, slotBits); err != nil {
			if v.emitAccountMetrics {
				metrics.ValidatorProposeFailVec.WithLabelValues(fmtKey).Inc()
			}
			return errors.Wrap(err, "failed to save updated proposal history")
		}
	}
	return nil
}
