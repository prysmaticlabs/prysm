package electra

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/altair"
	e "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"go.opencensus.io/trace"
)

// Re-exports for methods that haven't changed in Electra.
var (
	InitializePrecomputeValidators       = altair.InitializePrecomputeValidators
	ProcessEpochParticipation            = altair.ProcessEpochParticipation
	ProcessInactivityScores              = altair.ProcessInactivityScores
	ProcessRewardsAndPenaltiesPrecompute = altair.ProcessRewardsAndPenaltiesPrecompute
	ProcessSlashings                     = e.ProcessSlashings
	ProcessEth1DataReset                 = e.ProcessEth1DataReset
	ProcessSlashingsReset                = e.ProcessSlashingsReset
	ProcessRandaoMixesReset              = e.ProcessRandaoMixesReset
	ProcessHistoricalDataUpdate          = e.ProcessHistoricalDataUpdate
	ProcessParticipationFlagUpdates      = altair.ProcessParticipationFlagUpdates
	ProcessSyncCommitteeUpdates          = altair.ProcessSyncCommitteeUpdates
	AttestationsDelta                    = altair.AttestationsDelta
	ProcessSyncAggregate                 = altair.ProcessSyncAggregate
)

// ProcessEpoch describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
//
// Spec definition:
//
//	def process_epoch(state: BeaconState) -> None:
//	    process_justification_and_finalization(state)
//	    process_inactivity_updates(state)
//	    process_rewards_and_penalties(state)
//	    process_registry_updates(state)
//	    process_slashings(state)
//	    process_eth1_data_reset(state)
//	    process_pending_balance_deposits(state)  # New in EIP7251
//	    process_pending_consolidations(state)  # New in EIP7251
//	    process_effective_balance_updates(state)
//	    process_slashings_reset(state)
//	    process_randao_mixes_reset(state)
func ProcessEpoch(ctx context.Context, state state.BeaconState) error {
	_, span := trace.StartSpan(ctx, "electra.ProcessEpoch")
	defer span.End()

	if state == nil || state.IsNil() {
		return errors.New("nil state")
	}
	vp, bp, err := InitializePrecomputeValidators(ctx, state)
	if err != nil {
		return err
	}
	vp, bp, err = ProcessEpochParticipation(ctx, state, bp, vp)
	if err != nil {
		return err
	}
	state, err = precompute.ProcessJustificationAndFinalizationPreCompute(state, bp)
	if err != nil {
		return errors.Wrap(err, "could not process justification")
	}
	state, vp, err = ProcessInactivityScores(ctx, state, vp)
	if err != nil {
		return errors.Wrap(err, "could not process inactivity updates")
	}
	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		return errors.Wrap(err, "could not process rewards and penalties")
	}

	state, err = ProcessRegistryUpdates(ctx, state)
	if err != nil {
		return errors.Wrap(err, "could not process registry updates")
	}

	proportionalSlashingMultiplier, err := state.ProportionalSlashingMultiplier()
	if err != nil {
		return err
	}
	state, err = ProcessSlashings(state, proportionalSlashingMultiplier)
	if err != nil {
		return err
	}
	state, err = ProcessEth1DataReset(state)
	if err != nil {
		return err
	}

	if err = ProcessPendingBalanceDeposits(ctx, state, primitives.Gwei(bp.ActiveCurrentEpoch)); err != nil {
		return err
	}
	if err = ProcessPendingConsolidations(ctx, state); err != nil {
		return err
	}
	if err = ProcessEffectiveBalanceUpdates(state); err != nil {
		return err
	}

	state, err = ProcessSlashingsReset(state)
	if err != nil {
		return err
	}
	state, err = ProcessRandaoMixesReset(state)
	if err != nil {
		return err
	}
	state, err = ProcessHistoricalDataUpdate(state)
	if err != nil {
		return err
	}

	state, err = ProcessParticipationFlagUpdates(state)
	if err != nil {
		return err
	}

	_, err = ProcessSyncCommitteeUpdates(ctx, state)
	if err != nil {
		return err
	}

	return nil
}
