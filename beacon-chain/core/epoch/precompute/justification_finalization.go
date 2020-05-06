package precompute

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
)

// ProcessJustificationAndFinalizationPreCompute processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
// Note: this is an optimized version by passing in precomputed total and attesting balances.
func ProcessJustificationAndFinalizationPreCompute(state *stateTrie.BeaconState, pBal *Balance) (*stateTrie.BeaconState, error) {
	if state.Slot() <= helpers.StartSlot(2) {
		return state, nil
	}

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
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
	if 3*pBal.PrevEpochTargetAttested >= 2*pBal.ActiveCurrentEpoch {
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
	if 3*pBal.CurrentEpochTargetAttested >= 2*pBal.ActiveCurrentEpoch {
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

	// Process finalization according to ETH2.0 specifications.
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
