package stateGen

import (
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// generates state from the last finalized epoch till the specified slot.
func GenerateStateFromSlot(db *db.BeaconDB, slot uint64) (*pb.BeaconState, error) {
	fState, err := db.FinalizedState(slot)
	if err != nil {
		return nil, err
	}

	if fState.Slot >= slot {
		return nil, fmt.Errorf("requested slot is lower than or equal to the current slot in the finalized beacon state."+
			" Current finalized slot in state %d but was requested %d",
			fState.Slot, slot)
	}

	pBlock, err := db.BlockBySlot(fState.Slot)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve block: %v", err)
	}
	root, err := hashutil.HashBeaconBlock(pBlock)
	if err != nil {
		return nil, fmt.Errorf("could not tree hash parent block: %v", err)
	}

	// run N state transitions to generate state
	for i := fState.Slot + 1; i <= slot; i++ {
		exists, blk, err := db.HasBlockBySlot(i)
		if !exists {
			fState, err = state.ExecuteStateTransition(
				fState,
				nil,
				root,
				true, /* sig verify */
			)
			if err != nil {
				return nil, fmt.Errorf("could not execute state transition %v", err)
			}
			continue
		}

		fState, err = state.ExecuteStateTransition(
			fState,
			blk,
			root,
			true, /* sig verify */
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition %v", err)
		}

		root, err = hashutil.HashBeaconBlock(blk)
		if err != nil {
			return nil, fmt.Errorf("could not tree hash parent block: %v", err)
		}
	}

	return fState, nil
}
