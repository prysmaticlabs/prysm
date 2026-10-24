package eip8453

import (
	"context"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/electra"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/gloas"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/helpers"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/time"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/validators"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	fieldparams "github.com/OffchainLabs/prysm/v7/config/fieldparams"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
)

// ProcessEpoch runs the epoch transition, finality and rewards having moved to height events and rounds.
func ProcessEpoch(ctx context.Context, st state.BeaconState) error {
	if err := processRegistryUpdates(ctx, st); err != nil {
		return errors.Wrap(err, "could not process registry updates")
	}
	if err := electra.ProcessSlashings(ctx, st); err != nil {
		return errors.Wrap(err, "could not process slashings")
	}
	if err := ProcessPendingDeposits(ctx, st); err != nil {
		return errors.Wrap(err, "could not process pending deposits")
	}
	if err := electra.ProcessPendingConsolidations(ctx, st); err != nil {
		return errors.Wrap(err, "could not process pending consolidations")
	}
	if err := ProcessBuilderPendingPayments(st); err != nil {
		return errors.Wrap(err, "could not process builder pending payments")
	}
	if err := electra.ProcessEffectiveBalanceUpdates(st); err != nil {
		return errors.Wrap(err, "could not process effective balance updates")
	}
	st, err := electra.ProcessSlashingsReset(st)
	if err != nil {
		return err
	}
	st, err = electra.ProcessRandaoMixesReset(st)
	if err != nil {
		return err
	}
	st, err = electra.ProcessHistoricalDataUpdate(st)
	if err != nil {
		return err
	}
	if _, err := electra.ProcessSyncCommitteeUpdates(ctx, st); err != nil {
		return errors.Wrap(err, "could not process sync committee updates")
	}
	if err := gloas.ProcessProposerLookahead(ctx, st); err != nil {
		return errors.Wrap(err, "could not process proposer lookahead")
	}
	return processPTCWindow(st)
}

// Electra's version reads the Casper finalized checkpoint, EIP-8453 activates against the epoch of finalized_slot.
func processRegistryUpdates(ctx context.Context, st state.BeaconState) error {
	epoch := time.CurrentEpoch(st)
	finalized, err := finalizedEpoch(st)
	if err != nil {
		return err
	}
	cfg := params.BeaconConfig()
	activationEpoch := helpers.ActivationExitEpoch(epoch)
	for index := range primitives.ValidatorIndex(st.NumValidators()) {
		val, err := st.ValidatorAtIndexReadOnly(index)
		if err != nil {
			return err
		}
		switch {
		case val.ActivationEligibilityEpoch() == cfg.FarFutureEpoch && val.EffectiveBalance() >= cfg.MinActivationBalance:
			v := val.Copy()
			v.ActivationEligibilityEpoch = epoch + 1
			if err := st.UpdateValidatorAtIndex(index, v); err != nil {
				return err
			}
		case helpers.IsActiveValidatorUsingTrie(val, epoch) && val.EffectiveBalance() <= cfg.EjectionBalance:
			_, err := validators.InitiateValidatorExit(ctx, st, index, &validators.ExitInfo{})
			if err != nil && !errors.Is(err, validators.ErrValidatorAlreadyExited) {
				return err
			}
		case val.ActivationEligibilityEpoch() <= finalized && val.ActivationEpoch() == cfg.FarFutureEpoch:
			v := val.Copy()
			v.ActivationEpoch = activationEpoch
			if err := st.UpdateValidatorAtIndex(index, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Without beacon committees EIP-8453 samples each PTC from all active validators.
func processPTCWindow(st state.BeaconState) error {
	epoch := slots.ToEpoch(st.Slot()) + params.BeaconConfig().MinSeedLookahead + 1
	start, err := slots.EpochStart(epoch)
	if err != nil {
		return err
	}
	ptcs := make([]*ethpb.PTCs, params.BeaconConfig().SlotsPerEpoch)
	for i := range ptcs {
		ptc, err := slotCommittee(st, start+primitives.Slot(i), params.BeaconConfig().DomainPTCAttester, fieldparams.PTCSize)
		if err != nil {
			return err
		}
		ptcs[i] = &ethpb.PTCs{ValidatorIndices: ptc}
	}
	return st.RotatePTCWindow(ptcs)
}
