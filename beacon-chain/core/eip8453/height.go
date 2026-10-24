package eip8453

import (
	"context"
	"fmt"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	"github.com/OffchainLabs/prysm/v7/config/params"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/math"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
)

// ProcessHeightEvents finalizes the highest pending justification with a quorum, then justifies or times out the current height.
func ProcessHeightEvents(_ context.Context, st state.BeaconState) error {
	pendings, err := st.PendingFinalities()
	if err != nil {
		return err
	}
	finalized, err := st.FinalizedCheckpointEIP8453()
	if err != nil {
		return err
	}
	for _, pending := range pendings {
		quorum, err := hasQuorum(st, pending.Participation)
		if err != nil {
			return err
		}
		if !quorum {
			continue
		}
		finalized = pending.Checkpoint
		if err := st.SetFinalizedCheckpointEIP8453(pending.Checkpoint); err != nil {
			return err
		}
		if err := st.SetFinalizedSlot(pending.Slot); err != nil {
			return err
		}
	}
	retained := make([]*ethpb.PendingFinalityEIP8453, 0, len(pendings)+1)
	for _, pending := range pendings {
		if pending.Checkpoint.Height > finalized.Height {
			retained = append(retained, pending)
		}
	}
	if err := st.SetPendingFinalities(retained); err != nil {
		return err
	}

	targetVotes, err := st.TargetParticipationEIP8453()
	if err != nil {
		return err
	}
	justified, err := hasQuorum(st, targetVotes)
	if err != nil {
		return err
	}
	if justified {
		return justify(st, retained)
	}
	progress, err := st.ProgressParticipation()
	if err != nil {
		return err
	}
	quorum, err := hasQuorum(st, progress)
	if err != nil || !quorum {
		return err
	}
	targetSlot, err := st.TargetSlot()
	if err != nil {
		return err
	}
	timeoutSlot, err := math.Add64(uint64(targetSlot), uint64(params.BeaconConfig().SlotsPerRound))
	if err != nil {
		return err
	}
	if uint64(st.Slot()) < timeoutSlot {
		return nil
	}
	return advanceHeight(st)
}

func justify(st state.BeaconState, pendings []*ethpb.PendingFinalityEIP8453) error {
	target, err := st.TargetCheckpoint()
	if err != nil {
		return err
	}
	if err := st.SetJustifiedCheckpointEIP8453(target); err != nil {
		return err
	}
	targetSlot, err := st.TargetSlot()
	if err != nil {
		return err
	}
	if uint64(len(pendings)) == params.BeaconConfig().PendingFinalityWindow {
		pendings = pendings[1:]
	}
	pendings = append(pendings, &ethpb.PendingFinalityEIP8453{
		Checkpoint:    target,
		Slot:          targetSlot,
		Participation: make([]byte, st.NumValidators()),
	})
	if err := st.SetPendingFinalities(pendings); err != nil {
		return err
	}
	return advanceHeight(st)
}

// advanceHeight leaves the new target's root empty, the next process_slot fills it with this block's root.
func advanceHeight(st state.BeaconState) error {
	target, err := st.TargetCheckpoint()
	if err != nil {
		return err
	}
	if target.Height == ^primitives.Height(0) {
		return fmt.Errorf("height %d overflows", target.Height)
	}
	if err := st.SetTargetCheckpoint(&ethpb.HeightCheckpoint{Height: target.Height + 1, Root: make([]byte, 32)}); err != nil {
		return err
	}
	if err := st.SetTargetSlot(st.LatestBlockHeader().Slot); err != nil {
		return err
	}
	if err := st.SetTargetParticipationEIP8453(make([]byte, st.NumValidators())); err != nil {
		return err
	}
	return st.SetProgressParticipation(make([]byte, st.NumValidators()))
}
