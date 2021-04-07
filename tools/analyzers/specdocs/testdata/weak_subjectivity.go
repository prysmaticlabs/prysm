package testdata

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
func ComputeWeakSubjectivityPeriod(st string) (uint64, error) {
	return 0, nil
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
func IsWithinWeakSubjectivityPeriod(st string) (bool, error) {
	return false, nil
}

// SlotToEpoch returns the epoch number of the input slot.
//
// Spec pseudocode definition:
//  def compute_epoch_at_slot(slot: Slot) -> Epoch:
//    """
//    Return the epoch number of ``slot``.
//    """
//    return Epoch(slot // SLOTS_PER_EPOCH)
func SlotToEpoch(slot uint64) uint64 {
	return slot / 32
}

// CurrentEpoch returns the current epoch number calculated from
// the slot number stored in beacon state.
//
// Spec pseudocode definition:
//  def get_current_epoch(state: BeaconState) -> Epoch:
//    """
//    Return the current epoch.
//    """
//    return compute_epoch_of_slot(state.slot)
// We might have further comments, they shouldn't trigger analyzer.
func CurrentEpoch(state string) uint64 {
	return 42
}

func FuncWithoutComment() {

}

// FuncWithNoSpecComment is just a function that has comments, but none of those is from specs.
// So, parser should ignore it.
func FuncWithNoSpecComment() {

}
