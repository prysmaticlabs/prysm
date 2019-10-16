package precompute

import (
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// ProcessJustificationAndFinalizationPreCompute processes justification and finalization during
// epoch processing. This is where a beacon node can justify and finalize a new epoch.
// Note: this is an optimized version by passing in precomputed total and attesting balances.
func ProcessJustificationAndFinalizationPreCompute(state *pb.BeaconState, p *Balance) (*pb.BeaconState, error) {
	if state.Slot <= helpers.StartSlot(2) {
		return state, nil
	}

	prevEpoch := helpers.PrevEpoch(state)
	currentEpoch := helpers.CurrentEpoch(state)
	oldPrevJustifiedCheckpoint := state.PreviousJustifiedCheckpoint
	oldCurrJustifiedCheckpoint := state.CurrentJustifiedCheckpoint

	// Process justifications
	state.PreviousJustifiedCheckpoint = state.CurrentJustifiedCheckpoint
	state.JustificationBits.Shift(1)

	// Note: the spec refers to the bit index position starting at 1 instead of starting at zero.
	// We will use that paradigm here for consistency with the godoc spec definition.

	// If 2/3 or more of total balance attested in the previous epoch.
	if 3*p.PrevEpochTargetAttesters >= 2*p.CurrentEpoch {
		blockRoot, err := helpers.BlockRoot(state, prevEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for previous epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: prevEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(1, true)
	}

	// If 2/3 or more of the total balance attested in the current epoch.
	if 3*p.CurrentEpochTargetAttesters >= 2*p.CurrentEpoch {
		blockRoot, err := helpers.BlockRoot(state, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get block root for current epoch %d", prevEpoch)
		}
		state.CurrentJustifiedCheckpoint = &ethpb.Checkpoint{Epoch: currentEpoch, Root: blockRoot}
		state.JustificationBits.SetBitAt(0, true)
	}

	// Process finalization according to ETH2.0 specifications.
	justification := state.JustificationBits.Bytes()[0]

	// 2nd/3rd/4th (0b1110) most recent epochs are justified, the 2nd using the 4th as source.
	if justification&0x0E == 0x0E && (oldPrevJustifiedCheckpoint.Epoch+3) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 2nd/3rd (0b0110) most recent epochs are justified, the 2nd using the 3rd as source.
	if justification&0x06 == 0x06 && (oldPrevJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldPrevJustifiedCheckpoint
	}

	// 1st/2nd/3rd (0b0111) most recent epochs are justified, the 1st using the 3rd as source.
	if justification&0x07 == 0x07 && (oldCurrJustifiedCheckpoint.Epoch+2) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	// The 1st/2nd (0b0011) most recent epochs are justified, the 1st using the 2nd as source
	if justification&0x03 == 0x03 && (oldCurrJustifiedCheckpoint.Epoch+1) == currentEpoch {
		state.FinalizedCheckpoint = oldCurrJustifiedCheckpoint
	}

	return state, nil
}
