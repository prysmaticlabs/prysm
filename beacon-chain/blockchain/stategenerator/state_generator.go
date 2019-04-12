package stategenerator

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var log = logrus.WithField("prefix", "stategenerator")

// GenerateStateFromBlock generates state from the last finalized state to the input slot.
// Ex:
// 	1A - 2B(finalized) - 3C - 4 - 5D - 6 - 7F  (letters mean there's a block).
//  Input: slot 6.
//	Output: resulting state of state transition function after applying block C and D.
//  	along with skipped slot 4 and 6.
func GenerateStateFromBlock(ctx context.Context, db *db.BeaconDB, slot uint64) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.stategenerator.GenerateStateFromBlock")
	defer span.End()
	fState, err := db.HistoricalStateFromSlot(ctx, slot)
	if err != nil {
		return nil, err
	}

	// return finalized state if it's the same as input slot.
	if fState.Slot == slot {
		return fState, nil
	}

	// input slot can't be smaller than last finalized state's slot.
	if fState.Slot > slot {
		return nil, fmt.Errorf(
			"requested slot %d < current slot %d in the finalized beacon state",
			slot-params.BeaconConfig().GenesisSlot,
			fState.Slot-params.BeaconConfig().GenesisSlot,
		)
	}

	if fState.LatestBlock == nil {
		return nil, fmt.Errorf("latest head in state is nil %v", err)
	}

	fRoot, err := hashutil.HashBeaconBlock(fState.LatestBlock)
	if err != nil {
		return nil, fmt.Errorf("unable to get block root %v", err)
	}

	// from input slot, retrieve its corresponding block and call that the most recent block.
	mostRecentBlock, err := db.BlockBySlot(ctx, slot)
	if err != nil {
		return nil, err
	}

	// if the most recent block is a skip block, we get its parent block.
	// ex:
	// 	1A - 2B - 3C - 4 - 5 (letters mean there's a block).
	//  input slot is 5, but slots 4 and 5 are skipped, we get block C from slot 3.
	lastSlot := slot
	for mostRecentBlock == nil {
		lastSlot--
		mostRecentBlock, err = db.BlockBySlot(ctx, lastSlot)
		if err != nil {
			return nil, err
		}
	}

	// retrieve the block list to recompute state of the input slot.
	blocks, err := blocksSinceFinalized(ctx, db, mostRecentBlock, fRoot)
	if err != nil {
		return nil, fmt.Errorf("unable to look up block ancestors %v", err)
	}

	log.Infof("Recompute state starting last finalized slot %d and ending slot %d",
		fState.Slot-params.BeaconConfig().GenesisSlot, slot-params.BeaconConfig().GenesisSlot)
	postState := fState
	root := fRoot
	// this recomputes state up to the last available block.
	//	ex: 1A - 2B (finalized) - 3C - 4 - 5 - 6C - 7 - 8 (C is the last block).
	// 	input slot 8, this recomputes state to slot 6.
	for i := len(blocks); i > 0; i-- {
		block := blocks[i-1]
		if block.Slot <= postState.Slot {
			continue
		}
		// running state transitions for skipped slots.
		for block.Slot != fState.Slot+1 {
			postState, err = state.ExecuteStateTransition(
				ctx,
				postState,
				nil,
				root,
				&state.TransitionConfig{
					VerifySignatures: false,
					Logging:          false,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("could not execute state transition %v", err)
			}
		}
		postState, err = state.ExecuteStateTransition(
			ctx,
			postState,
			block,
			root,
			&state.TransitionConfig{
				VerifySignatures: false,
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

	// this recomputes state from last block to last slot if there's skipp slots after.
	//	ex: 1A - 2B (finalized) - 3C - 4 - 5 - 6C - 7 - 8 (7 and 8 are skipped slots).
	// 	input slot 8, this recomputes state from 6C to 8.
	for i := postState.Slot; i < slot; i++ {
		postState, err = state.ExecuteStateTransition(
			ctx,
			postState,
			nil,
			root,
			&state.TransitionConfig{
				VerifySignatures: false,
				Logging:          false,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not execute state transition %v", err)
		}
	}

	log.Infof("Finished recompute state with slot %d and finalized epoch %d",
		postState.Slot-params.BeaconConfig().GenesisSlot, postState.FinalizedEpoch-params.BeaconConfig().GenesisEpoch)

	return postState, nil
}

// blocksSinceFinalized will return a list of linked blocks that's
// between the input block and the last finalized block in the db.
// The input block is also returned in the list.
// Ex:
// 	A -> B(finalized) -> C -> D -> E -> D.
// 	Input: E, output: [E, D, C, B].
func blocksSinceFinalized(ctx context.Context, db *db.BeaconDB, block *pb.BeaconBlock,
	finalizedBlockRoot [32]byte) ([]*pb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.stategenerator.blocksSinceFinalized")
	defer span.End()
	blockAncestors := make([]*pb.BeaconBlock, 0)
	blockAncestors = append(blockAncestors, block)
	parentRoot := bytesutil.ToBytes32(block.ParentRootHash32)
	// looking up ancestors, until the finalized block.
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
