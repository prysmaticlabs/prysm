package helpers

import (
	"bytes"
	"errors"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ComputeWeakSubjectivityPeriod returns weak subjectivity period for the active validator count and finalized epoch.
//
// Reference spec implementation:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
//
// def compute_weak_subjectivity_period(state: BeaconState) -> uint64:
//    """
//    Returns the weak subjectivity period for the current ``state``.
//    This computation takes into account the effect of:
//        - validator set churn (bounded by ``get_validator_churn_limit()`` per epoch), and
//        - validator balance top-ups (bounded by ``MAX_DEPOSITS * SLOTS_PER_EPOCH`` per epoch).
//    A detailed calculation can be found at:
//    https://github.com/runtimeverification/beacon-chain-verification/blob/master/weak-subjectivity/weak-subjectivity-analysis.pdf
//    """
//    ws_period = MIN_VALIDATOR_WITHDRAWABILITY_DELAY
//    N = len(get_active_validator_indices(state, get_current_epoch(state)))
//    t = get_total_active_balance(state) // N // ETH_TO_GWEI
//    T = MAX_EFFECTIVE_BALANCE // ETH_TO_GWEI
//    delta = get_validator_churn_limit(state)
//    Delta = MAX_DEPOSITS * SLOTS_PER_EPOCH
//    D = SAFETY_DECAY
//
//    if T * (200 + 3 * D) < t * (200 + 12 * D):
//        epochs_for_validator_set_churn = (
//            N * (t * (200 + 12 * D) - T * (200 + 3 * D)) // (600 * delta * (2 * t + T))
//        )
//        epochs_for_balance_top_ups = (
//            N * (200 + 3 * D) // (600 * Delta)
//        )
//        ws_period += max(epochs_for_validator_set_churn, epochs_for_balance_top_ups)
//    else:
//        ws_period += (
//            3 * N * D * t // (200 * Delta * (T - t))
//        )
//
//    return ws_period
func ComputeWeakSubjectivityPeriod(st iface.ReadOnlyBeaconState) (types.Epoch, error) {
	// Weak subjectivity period cannot be smaller than withdrawal delay.
	wsp := uint64(params.BeaconConfig().MinValidatorWithdrawabilityDelay)

	// Cardinality of active validator set.
	N, err := ActiveValidatorCount(st, CurrentEpoch(st))
	if err != nil {
		return 0, fmt.Errorf("cannot obtain active valiadtor count: %w", err)
	}
	if N == 0 {
		return 0, errors.New("no active validators found")
	}

	// Average effective balance in the given validator set, in Ether.
	t, err := TotalActiveBalance(st)
	if err != nil {
		return 0, fmt.Errorf("cannot find total active balance of validators: %w", err)
	}
	t = t / N / params.BeaconConfig().GweiPerEth

	// Maximum effective balance per validator.
	T := params.BeaconConfig().MaxEffectiveBalance / params.BeaconConfig().GweiPerEth

	// Validator churn limit.
	delta, err := ValidatorChurnLimit(N)
	if err != nil {
		return 0, fmt.Errorf("cannot obtain active validator churn limit: %w", err)
	}

	// Balance top-ups.
	Delta := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().MaxDeposits))

	if delta == 0 || Delta == 0 {
		return 0, errors.New("either validator churn limit or balance top-ups is zero")
	}

	// Safety decay, maximum tolerable loss of safety margin of FFG finality.
	D := params.BeaconConfig().SafetyDecay

	if T*(200+3*D) < t*(200+12*D) {
		epochsForValidatorSetChurn := N * (t*(200+12*D) - T*(200+3*D)) / (600 * delta * (2*t + T))
		epochsForBalanceTopUps := N * (200 + 3*D) / (600 * Delta)
		wsp += mathutil.Max(epochsForValidatorSetChurn, epochsForBalanceTopUps)
	} else {
		wsp += 3 * N * D * t / (200 * Delta * (T - t))
	}

	return types.Epoch(wsp), nil
}

// IsWithinWeakSubjectivityPeriod verifies if a given weak subjectivity checkpoint is not stale i.e.
// the current node is so far beyond, that a given state and checkpoint are not for the latest weak
// subjectivity point. Provided checkpoint still can be used to double-check that node's block root
// at a given epoch matches that of the checkpoint.
//
// Reference implementation:
// https://github.com/ethereum/eth2.0-specs/blob/master/specs/phase0/weak-subjectivity.md#checking-for-stale-weak-subjectivity-checkpoint

// def is_within_weak_subjectivity_period(store: Store, ws_state: BeaconState, ws_checkpoint: Checkpoint) -> bool:
//    # Clients may choose to validate the input state against the input Weak Subjectivity Checkpoint
//    assert ws_state.latest_block_header.state_root == ws_checkpoint.root
//    assert compute_epoch_at_slot(ws_state.slot) == ws_checkpoint.epoch
//
//    ws_period = compute_weak_subjectivity_period(ws_state)
//    ws_state_epoch = compute_epoch_at_slot(ws_state.slot)
//    current_epoch = compute_epoch_at_slot(get_current_slot(store))
//    return current_epoch <= ws_state_epoch + ws_period
func IsWithinWeakSubjectivityPeriod(
	currentEpoch types.Epoch, wsState iface.ReadOnlyBeaconState, wsCheckpoint *eth.WeakSubjectivityCheckpoint) (bool, error) {
	// Make sure that incoming objects are not nil.
	if wsState == nil || wsState.LatestBlockHeader() == nil || wsCheckpoint == nil {
		return false, errors.New("invalid weak subjectivity state or checkpoint")
	}

	// Assert that state and checkpoint have the same root and epoch.
	if !bytes.Equal(wsState.LatestBlockHeader().StateRoot, wsCheckpoint.StateRoot) {
		return false, fmt.Errorf("state (%#x) and checkpoint (%#x) roots do not match",
			wsState.LatestBlockHeader().StateRoot, wsCheckpoint.StateRoot)
	}
	if SlotToEpoch(wsState.Slot()) != wsCheckpoint.Epoch {
		return false, fmt.Errorf("state (%v) and checkpoint (%v) epochs do not match",
			SlotToEpoch(wsState.Slot()), wsCheckpoint.Epoch)
	}

	// Compare given epoch to state epoch + weak subjectivity period.
	wsPeriod, err := ComputeWeakSubjectivityPeriod(wsState)
	if err != nil {
		return false, fmt.Errorf("cannot compute weak subjectivity period: %w", err)
	}
	wsStateEpoch := SlotToEpoch(wsState.Slot())

	return currentEpoch <= wsStateEpoch+wsPeriod, nil
}
