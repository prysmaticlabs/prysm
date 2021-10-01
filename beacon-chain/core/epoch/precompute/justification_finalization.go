package precompute

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// ProcessJustificationAndFinalizationPreCompute processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
// Note: this is an optimized version by passing in precomputed total and attesting balances.
//
// Spec pseudocode definition:
//  def process_justification_and_finalization(state: BeaconState) -> None:
//    # Initial FFG checkpoint values have a `0x00` stub for `root`.
//    # Skip FFG updates in the first two epochs to avoid corner cases that might result in modifying this stub.
//    if get_current_epoch(state) <= GENESIS_EPOCH + 1:
//        return
//    previous_attestations = get_matching_target_attestations(state, get_previous_epoch(state))
//    current_attestations = get_matching_target_attestations(state, get_current_epoch(state))
//    total_active_balance = get_total_active_balance(state)
//    previous_target_balance = get_attesting_balance(state, previous_attestations)
//    current_target_balance = get_attesting_balance(state, current_attestations)
//    weigh_justification_and_finalization(state, total_active_balance, previous_target_balance, current_target_balance)
func ProcessJustificationAndFinalizationPreCompute(state state.BeaconState, pBal *Balance) (state.BeaconState, error) {
	canProcessSlot, err := slots.EpochStart(2 /*epoch*/)
	if err != nil {
		return nil, err
	}
	if state.Slot() <= canProcessSlot {
		return state, nil
	}

	return weighJustificationAndFinalization(state, pBal.ActiveCurrentEpoch, pBal.PrevEpochTargetAttested, pBal.CurrentEpochTargetAttested)
}

// weighJustificationAndFinalization processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
//
// Spec pseudocode definition:
// def weigh_justification_and_finalization(state: BeaconState,
//                                         total_active_balance: Gwei,
//                                         previous_epoch_target_balance: Gwei,
//                                         current_epoch_target_balance: Gwei) -> None:
//    previous_epoch = get_previous_epoch(state)
//    current_epoch = get_current_epoch(state)
//    old_previous_justified_checkpoint = state.previous_justified_checkpoint
//    old_current_justified_checkpoint = state.current_justified_checkpoint
//
//    # Process justifications
//    state.previous_justified_checkpoint = state.current_justified_checkpoint
//    state.justification_bits[1:] = state.justification_bits[:JUSTIFICATION_BITS_LENGTH - 1]
//    state.justification_bits[0] = 0b0
//    if previous_epoch_target_balance * 3 >= total_active_balance * 2:
//        state.current_justified_checkpoint = Checkpoint(epoch=previous_epoch,
//                                                        root=get_block_root(state, previous_epoch))
//        state.justification_bits[1] = 0b1
//    if current_epoch_target_balance * 3 >= total_active_balance * 2:
//        state.current_justified_checkpoint = Checkpoint(epoch=current_epoch,
//                                                        root=get_block_root(state, current_epoch))
//        state.justification_bits[0] = 0b1
//
//    # Process finalizations
//    bits = state.justification_bits
//    # The 2nd/3rd/4th most recent epochs are justified, the 2nd using the 4th as source
//    if all(bits[1:4]) and old_previous_justified_checkpoint.epoch + 3 == current_epoch:
//        state.finalized_checkpoint = old_previous_justified_checkpoint
//    # The 2nd/3rd most recent epochs are justified, the 2nd using the 3rd as source
//    if all(bits[1:3]) and old_previous_justified_checkpoint.epoch + 2 == current_epoch:
//        state.finalized_checkpoint = old_previous_justified_checkpoint
//    # The 1st/2nd/3rd most recent epochs are justified, the 1st using the 3rd as source
//    if all(bits[0:3]) and old_current_justified_checkpoint.epoch + 2 == current_epoch:
//        state.finalized_checkpoint = old_current_justified_checkpoint
//    # The 1st/2nd most recent epochs are justified, the 1st using the 2nd as source
//    if all(bits[0:2]) and old_current_justified_checkpoint.epoch + 1 == current_epoch:
//        state.finalized_checkpoint = old_current_justified_checkpoint
func weighJustificationAndFinalization(state state.BeaconState,
	totalActiveBalance, prevEpochTargetBalance, currEpochTargetBalance uint64) (state.BeaconState, error) {
	prevEpoch := time.PrevEpoch(state)
	currentEpoch := time.CurrentEpoch(state)
	oldPrevJustifiedCheckpoint := state.PreviousJustifiedCheckpoint()
	oldCurrJustifiedCheckpoint := state.CurrentJustifiedCheckpoint()

	// Process justifications
	if err := state.SetPreviousJustifiedCheckpoint(state.CurrentJustifiedCheckpoint()); err != nil {
		return nil, err
	}
	newBits := state.JustificationBits()
	newBits.Shift(1)
	if err := state.SetJustificationBits(newBits); err != nil {
		return nil, err
	}

	// Note: the spec refers to the bit index position starting at 1 instead of starting at zero.
	// We will use that paradigm here for consistency with the godoc spec definition.

	// If 2/3 or more of total balance attested in the previous epoch.
	if 3*prevEpochTargetBalance >= 2*totalActiveBalance {
		blockRoot, err := helpers.BlockRoot(state, prevEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for previous epoch %d", prevEpoch)
		}
		if err := state.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: prevEpoch, Root: blockRoot}); err != nil {
			return nil, err
		}
		newBits := state.JustificationBits()
		newBits.SetBitAt(1, true)
		if err := state.SetJustificationBits(newBits); err != nil {
			return nil, err
		}
	}

	// If 2/3 or more of the total balance attested in the current epoch.
	if 3*currEpochTargetBalance >= 2*totalActiveBalance {
		blockRoot, err := helpers.BlockRoot(state, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for current epoch %d", prevEpoch)
		}
		if err := state.SetCurrentJustifiedCheckpoint(&ethpb.Checkpoint{Epoch: currentEpoch, Root: blockRoot}); err != nil {
			return nil, err
		}
		newBits := state.JustificationBits()
		newBits.SetBitAt(0, true)
		if err := state.SetJustificationBits(newBits); err != nil {
			return nil, err
		}
	}

	// Process finalization according to Ethereum Beacon Chain specification.
	justification := state.JustificationBits().Bytes()[0]

	// 2nd/3rd/4th (0b1110) most recent epochs are justified, the 2nd using the 4th as source.
	if justification&0x0E == 0x0E && (oldPrevJustifiedCheckpoint.Epoch+3) == currentEpoch {
		if err := state.SetFinalizedCheckpoint(oldPrevJustifiedCheckpoint); err != nil {
			return nil, err
		}
	}

	// 2nd/3rd (0b0110) most recent epochs are justified, the 2nd using the 3rd as source.
	if justification&0x06 == 0x06 && (oldPrevJustifiedCheckpoint.Epoch+2) == currentEpoch {
		if err := state.SetFinalizedCheckpoint(oldPrevJustifiedCheckpoint); err != nil {
			return nil, err
		}
	}

	// 1st/2nd/3rd (0b0111) most recent epochs are justified, the 1st using the 3rd as source.
	if justification&0x07 == 0x07 && (oldCurrJustifiedCheckpoint.Epoch+2) == currentEpoch {
		if err := state.SetFinalizedCheckpoint(oldCurrJustifiedCheckpoint); err != nil {
			return nil, err
		}
	}

	// The 1st/2nd (0b0011) most recent epochs are justified, the 1st using the 2nd as source
	if justification&0x03 == 0x03 && (oldCurrJustifiedCheckpoint.Epoch+1) == currentEpoch {
		if err := state.SetFinalizedCheckpoint(oldCurrJustifiedCheckpoint); err != nil {
			return nil, err
		}
	}

	return state, nil
}
