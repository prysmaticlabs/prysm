package kv

import (
	"context"
	"fmt"
	"runtime"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	transition "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Using max possible size to avoid using DB to save and retrieve pre state (slow)
// The size is 80 because block at slot 43772 built on top of block at slot 43693.
// That is the worst case.
const historicalStatesSize = 80

func (kv *Store) regenHistoricalStates(ctx context.Context) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.regenHistoricalStates")
	defer span.End()

	genesisState, err := kv.GenesisState(ctx)
	if err != nil {
		return err
	}
	currentState := genesisState.Copy()
	startSlot := genesisState.Slot()

	// Restore from last archived point if this process was previously interrupted.
	slotsPerArchivedPoint := params.BeaconConfig().SlotsPerArchivedPoint
	lastArchivedIndex, err := kv.LastArchivedIndex(ctx)
	if err != nil {
		return err
	}
	if lastArchivedIndex > 0 {
		archivedIndexStart := lastArchivedIndex - 1
		archivedRoot := kv.ArchivedPointRoot(ctx, archivedIndexStart)
		currentState, err := kv.State(ctx, archivedRoot)
		if err != nil {
			return err
		}
		startSlot = currentState.Slot()
	}

	lastSavedBlockArchivedIndex, err := kv.lastSavedBlockArchivedIndex(ctx)
	if err != nil {
		return err
	}

	cacheState, err := lru.New(historicalStatesSize)
	if err != nil {
		return err
	}
	for i := lastArchivedIndex; i <= lastSavedBlockArchivedIndex; i++ {
		// This is an expensive operation, so we check if the context was canceled
		// at any point in the iteration.
		if err := ctx.Err(); err != nil {
			return err
		}
		targetSlot := startSlot + slotsPerArchivedPoint
		filter := filters.NewFilter().SetStartSlot(startSlot + 1).SetEndSlot(targetSlot)
		blocks, err := kv.Blocks(ctx, filter)
		if err != nil {
			return err
		}

		// Replay blocks and replay slots if necessary.
		if len(blocks) > 0 {
			for i := 0; i < len(blocks); i++ {
				if blocks[i].Block.Slot == 0 {
					continue
				}

				var preState *stateTrie.BeaconState
				item, ok := cacheState.Get(bytesutil.ToBytes32(blocks[i].Block.ParentRoot))
				if !ok {
					preState, err = kv.State(ctx, bytesutil.ToBytes32(blocks[i].Block.ParentRoot))
					if err != nil {
						return err
					}
				} else {
					preState = item.(*stateTrie.BeaconState).Copy()
				}
				if preState == nil {
					return errors.New("pre state can't be nil")
				}

				currentState, err = regenHistoricalStateTransition(ctx, preState.Copy(), blocks[i])
				if err != nil {
					return errors.Wrap(err, "could not regenerate historical state transition")
				}

				r, err := stateutil.BlockRoot(blocks[i].Block)
				if err != nil {
					return err
				}
				cacheState.Add(r, currentState)
			}
		}
		if targetSlot > currentState.Slot() {
			currentState, err = regenHistoricalStateProcessSlots(ctx, currentState, targetSlot)
			if err != nil {
				return errors.Wrap(err, "could not regenerate historical process slot")
			}
		}

		if len(blocks) > 0 {
			// Save the historical root, state and highest index to the DB.
			if currentState.Slot()%slotsPerArchivedPoint == 0 {
				if err := kv.saveArchivedInfo(ctx, currentState, blocks); err != nil {
					return err
				}
				log.WithFields(log.Fields{
					"currentArchivedIndex/totalArchivedIndices": fmt.Sprintf("%d/%d", i, lastSavedBlockArchivedIndex),
					"archivedStateSlot":                         currentState.Slot()}).Info("Saved historical state")
			}
		}
		startSlot += slotsPerArchivedPoint
	}

	// Flush the cache, the cached states never be used again.
	cacheState.Purge()

	// Manually garbage collect as previous cache will never be used again.
	runtime.GC()

	return nil
}

// This runs state transition to recompute historical state.
func regenHistoricalStateTransition(
	ctx context.Context,
	state *stateTrie.BeaconState,
	signed *ethpb.SignedBeaconBlock,
) (*stateTrie.BeaconState, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if signed == nil || signed.Block == nil {
		return nil, errors.New("block can't be nil")
	}
	ctx, span := trace.StartSpan(ctx, "BeaconDB.regenHistoricalStateTransition")
	defer span.End()
	var err error
	state, err = regenHistoricalStateProcessSlots(ctx, state, signed.Block.Slot)
	if err != nil {
		return nil, errors.Wrap(err, "could not process slot")
	}
	state, err = transition.ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block")
	}
	return state, nil
}

// This runs slot transition to recompute historical state.
func regenHistoricalStateProcessSlots(ctx context.Context, state *stateTrie.BeaconState, slot uint64) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.regenHistoricalStateProcessSlots")
	defer span.End()
	if state == nil {
		return nil, errors.New("state can't be nil")
	}
	if state.Slot() > slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot(), slot)
		return nil, err
	}
	if state.Slot() == slot {
		return state, nil
	}
	for state.Slot() < slot {
		state, err := transition.ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if transition.CanProcessEpoch(state) {
			state, err = transition.ProcessEpochPrecompute(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		if err := state.SetSlot(state.Slot() + 1); err != nil {
			return nil, err
		}
	}
	return state, nil
}

// This retrieves the last saved block's archived index.
func (kv *Store) lastSavedBlockArchivedIndex(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.lastSavedBlockArchivedIndex")
	defer span.End()
	b, err := kv.HighestSlotBlocks(ctx)
	if err != nil {
		return 0, err
	}
	if len(b) == 0 {
		return 0, errors.New("blocks can't be empty")
	}
	if b[0] == nil {
		return 0, errors.New("nil last block")
	}
	lastSavedBlockSlot := b[0].Block.Slot
	slotsPerArchivedPoint := params.BeaconConfig().SlotsPerArchivedPoint
	lastSavedBlockArchivedIndex := lastSavedBlockSlot/slotsPerArchivedPoint - 1

	return lastSavedBlockArchivedIndex, nil
}

// This saved archived info (state, root) into the db.
func (kv *Store) saveArchivedInfo(ctx context.Context,
	currentState *stateTrie.BeaconState,
	blocks []*ethpb.SignedBeaconBlock,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.saveArchivedInfo")
	defer span.End()
	span.AddAttributes(trace.Int64Attribute("slot", int64(currentState.Slot())))

	if len(blocks) == 0 {
		return errors.New("no blocks provided")
	}

	lastBlocksRoot, err := stateutil.BlockRoot(blocks[len(blocks)-1].Block)
	if err != nil {
		return nil
	}
	if err := kv.SaveState(ctx, currentState, lastBlocksRoot); err != nil {
		return err
	}
	if err := kv.SaveArchivedPointRoot(ctx, lastBlocksRoot, currentState.Slot()); err != nil {
		return err
	}
	return nil
}
