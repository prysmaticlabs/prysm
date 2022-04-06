package helpers

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/math"
	v1alpha1 "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

var errWeakSubjectivityBalanceUnavailable = errors.New("cannot find total active balance of validators")
var errWeakSubjectivityValidatorCountUnavailable = errors.New("cannot obtain active validator count")
var errWeakSubjectivityChurnLimit = errors.New("cannot obtain active validator churn limit")
var errWeakSubjectivityZeroChurn = errors.New("either validator churn limit, or max deposits per-epoch, is zero")

// these are exported for tests in helpers_test (see export_test.go)
var errWeakSubjectivityZeroValidators = errors.New("cannot compute weak subjectivity with 0 active validators")
var errInvalidWeakSubjectivityState = errors.New("invalid weak subjectivity state or checkpoint")
var errWeakSubjectivityMismatchedRoot = errors.New("state and checkpoint roots do not match")
var errWeakSubjectivityMismatchedEpoch = errors.New("state and checkpoint epochs do not match")

// CurrentWeakSubjectivityPeriod returns weak subjectivity period for a given state.
func CurrentWeakSubjectivityPeriod(ctx context.Context, st state.ReadOnlyBeaconState, cfg *params.BeaconChainConfig) (types.Epoch, error) {
	activeValidators, err := ActiveValidatorCount(ctx, st, time.CurrentEpoch(st))
	if err != nil {
		return 0, errors.Wrap(errWeakSubjectivityValidatorCountUnavailable, err.Error())
	}
	totalActiveBalance, err := TotalActiveBalance(st)
	if err != nil {
		return 0, errors.Wrap(errWeakSubjectivityBalanceUnavailable, err.Error())
	}
	return ComputeWeakSubjectivityPeriod(cfg, activeValidators, totalActiveBalance)
}

// ComputeWeakSubjectivityPeriod is an implementation of the below reference spec. It uses explicit parameters for
// values derived from the state for ease of verification and experimentation.
//
// https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#calculating-the-weak-subjectivity-period
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
func ComputeWeakSubjectivityPeriod(cfg *params.BeaconChainConfig, activeValidators, totalActiveBalance uint64) (types.Epoch, error) {
	// The Weak Subjectivity Period should never be less than MIN_VALIDATOR_WITHDRAWABILITY_DELAY.
	// To ensure this, the results of the computation are added to MinValidatorWithdrawabilityDelay.
	wsPeriod := cfg.MinValidatorWithdrawabilityDelay

	if activeValidators == 0 {
		return 0, errWeakSubjectivityZeroValidators
	}
	// Variables like 'N' are used to 1:1 match the spec. N is the total number of active validators.
	N := activeValidators

	// Total active balance is denominated in Gwei. Dividing it by N (count of active validators)
	// gives the average balance per Validator in Gwei. All computations are performed in Ether, rather than Gwei,
	// to reduce the likelihood of an overflow. The division by GweiPerEth is the conversion to Ether.
	t := totalActiveBalance / N / cfg.GweiPerEth

	// Maximum effective balance per Validator, also converted to Ether.
	T := cfg.MaxEffectiveBalance / cfg.GweiPerEth

	// delta is the Validator churn limit.
	delta, err := ValidatorChurnLimit(activeValidators)
	if err != nil {
		return 0, errors.Wrap(errWeakSubjectivityChurnLimit, err.Error())
	}
	// Delta is the upper bound on how many Validators can deposit in an Epoch, in other words,
	// it sets the entry rate limit.
	// Notice the pattern of lowercase/uppercase variable names ratios of actual/max: delta/Delta, t/T
	Delta := uint64(cfg.SlotsPerEpoch.Mul(cfg.MaxDeposits))

	if delta == 0 || Delta == 0 {
		return 0, errWeakSubjectivityZeroChurn
	}

	// Safety decay, maximum tolerable loss of safety margin of FFG finality.
	D := cfg.SafetyDecay

	if T*(200+3*D) < t*(200+12*D) {
		epochsForValidatorSetChurn := N * (t*(200+12*D) - T*(200+3*D)) / (600 * delta * (2*t + T))
		epochsForBalanceTopUps := N * (200 + 3*D) / (600 * Delta)
		return wsPeriod + types.Epoch(math.Max(epochsForValidatorSetChurn, epochsForBalanceTopUps)), nil
	}
	return wsPeriod + types.Epoch(3*N*D*t/(200*Delta*(T-t))), nil
}

// IsWithinWeakSubjectivityPeriod verifies if a given weak subjectivity checkpoint is not stale i.e.
// the current node is so far beyond, that a given state and checkpoint are not for the latest weak
// subjectivity point. Provided checkpoint still can be used to double-check that node's block root
// at a given epoch matches that of the checkpoint.
//
// Reference implementation:
// https://github.com/ethereum/consensus-specs/blob/master/specs/phase0/weak-subjectivity.md#checking-for-stale-weak-subjectivity-checkpoint
//
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
	ctx context.Context, currentEpoch types.Epoch, wsState state.ReadOnlyBeaconState, wsStateRoot [fieldparams.RootLength]byte, wsEpoch types.Epoch, cfg *params.BeaconChainConfig) (bool, error) {
	// Make sure that incoming objects are not nil.
	if wsState == nil || wsState.IsNil() || wsState.LatestBlockHeader() == nil {
		return false, errInvalidWeakSubjectivityState
	}

	hr := bytesutil.ToBytes32(wsState.LatestBlockHeader().StateRoot)
	// Assert that state and checkpoint have the same root and epoch.
	if hr != wsStateRoot {
		return false, errors.Wrapf(errWeakSubjectivityMismatchedRoot, "state=%#x, checkpoint=%#x", hr, wsStateRoot)
	}

	se := slots.ToEpoch(wsState.Slot())
	if se != wsEpoch {
		return false, errors.Wrapf(errWeakSubjectivityMismatchedEpoch, "state=%v, checkpoint=%v", se, wsEpoch)
	}

	// Compare given epoch to state epoch + weak subjectivity period.
	wsPeriod, err := CurrentWeakSubjectivityPeriod(ctx, wsState, cfg)
	if err != nil {
		return false, errors.Wrapf(err, "cannot check if within weak subjectivity period")
	}

	return currentEpoch <= se+wsPeriod, nil
}

// LatestWeakSubjectivityEpoch returns epoch of the most recent weak subjectivity checkpoint known to a node.
//
// Within the weak subjectivity period, if two conflicting blocks are finalized, 1/3 - D (D := safety decay)
// of validators will get slashed. Therefore, it is safe to assume that any finalized checkpoint within that
// period is protected by this safety margin.
func LatestWeakSubjectivityEpoch(ctx context.Context, st state.ReadOnlyBeaconState, cfg *params.BeaconChainConfig) (types.Epoch, error) {
	wsPeriod, err := CurrentWeakSubjectivityPeriod(ctx, st, cfg)
	if err != nil {
		return 0, err
	}

	finalizedEpoch := st.FinalizedCheckpointEpoch()
	return finalizedEpoch - (finalizedEpoch % wsPeriod), nil
}

// ParseWeakSubjectivityInputString parses "blocks_root:epoch_number" string into a checkpoint.
func ParseWeakSubjectivityInputString(wsCheckpointString string) (*v1alpha1.Checkpoint, error) {
	if wsCheckpointString == "" {
		return nil, nil
	}

	// Weak subjectivity input string must contain ":" to separate epoch and block root.
	if !strings.Contains(wsCheckpointString, ":") {
		return nil, fmt.Errorf("%s did not contain column", wsCheckpointString)
	}

	// Strip prefix "0x" if it's part of the input string.
	wsCheckpointString = strings.TrimPrefix(wsCheckpointString, "0x")

	// Get the hexadecimal block root from input string.
	s := strings.Split(wsCheckpointString, ":")
	if len(s) != 2 {
		return nil, errors.New("weak subjectivity checkpoint input should be in `block_root:epoch_number` format")
	}

	bRoot, err := hex.DecodeString(s[0])
	if err != nil {
		return nil, err
	}
	if len(bRoot) != 32 {
		return nil, errors.New("block root is not length of 32")
	}

	// Get the epoch number from input string.
	epoch, err := strconv.ParseUint(s[1], 10, 64)
	if err != nil {
		return nil, err
	}

	return &v1alpha1.Checkpoint{
		Epoch: types.Epoch(epoch),
		Root:  bRoot,
	}, nil
}
