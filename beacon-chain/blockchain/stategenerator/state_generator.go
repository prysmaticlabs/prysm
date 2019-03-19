package stategenerator

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("prefix", "stategenerator")

// GenerateStateFromBlock generates state from the historical state at the
// given block's slot.
func GenerateStateFromBlock(ctx context.Context, db *db.BeaconDB, block *pb.BeaconBlock) (*pb.BeaconState, error) {
	hState, err := db.HistoricalStateFromSlot(block.Slot)
	if err != nil {
		return nil, err
	}
	if hState.Slot == block.Slot {
		return hState, nil
	}

	if hState.Slot > block.Slot {
		return nil, fmt.Errorf(
			"requested slot %d < current slot %d in the historical beacon state",
			hState.Slot,
			block,
		)
	}

	root, err := hashutil.HashBeaconBlock(hState.LatestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to get block root %v", err)
	}

	ancestorSet, err := ancestersToLastFinalizedBlock(db, block, root)
	if err != nil {
		return nil, fmt.Errorf("unable to look up block ancestors %v", err)
	}

	log.Debugf("Start: Current slot %d and Finalized Epoch %d", hState.Slot, hState.FinalizedEpoch)

	for i := len(ancestorSet); i > 0; i-- {
		block := ancestorSet[i-1]

		if block.Slot <= hState.Slot {
			continue
		}
		// Running state transitions for skipped slots.
		for block.Slot != hState.Slot+1 {
			hState, err = state.ExecuteStateTransition(
				ctx,
				hState,
				nil,
				root,
				&state.TransitionConfig{
					VerifySignatures: true,
					Logging:          false,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("could not execute state transition %v", err)
			}
		}

		hState, err = state.ExecuteStateTransition(
			ctx,
			hState,
			block,
			root,
			&state.TransitionConfig{
				VerifySignatures: true,
				Logging:          false,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition %v", err)
		}

		root, err = hashutil.HashBeaconBlock(block)
		if err != nil {
			return nil, fmt.Errorf("unable to get block root %v", err)
		}
	}

	log.Debugf("End: Current slot %d and Finalized Epoch %d", hState.Slot, hState.FinalizedEpoch)

	return hState, nil
}

// ancestersToLastFinalizedBlock will return a list of all block ancesters
// between the given block and the most recent finalized block in the db.
// The given block is also returned in the list of ancesters.
func ancestersToLastFinalizedBlock(db *db.BeaconDB, block *pb.BeaconBlock,
	finalizedBlockRoot [32]byte) ([]*pb.BeaconBlock, error) {

	blockAncestors := make([]*pb.BeaconBlock, 0)
	blockAncestors = append(blockAncestors, block)

	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	// looking up ancestors, till it gets the last finalized block.
	for parentRoot != finalizedBlockRoot {
		retblock, err := db.Block(parentRoot)
		if err != nil {
			return nil, err
		}
		blockAncestors = append(blockAncestors, retblock)
		parentRoot = bytesutil.ToBytes32(retblock.ParentRootHash32)
	}

	return blockAncestors, nil
}
