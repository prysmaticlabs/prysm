package altair

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	coreState "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// SkipSlotCache exists for the unlikely scenario that is a large gap between the head state and
// the current slot. If the beacon chain were ever to be stalled for several epochs, it may be
// difficult or impossible to compute the appropriate beacon state for assignments within a
// reasonable amount of time.
var SkipSlotCache = cache.NewSkipSlotCache()

// CalculateStateRoot is used for calculating the
// state root of the state for the block proposer to use.
// This does not modify state.
func CalculateStateRoot(
	ctx context.Context,
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlockAltair,
) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "altair.CalculateStateRoot")
	defer span.End()
	if ctx.Err() != nil {
		return [32]byte{}, ctx.Err()
	}
	if state == nil {
		return [32]byte{}, errors.New("nil state")
	}
	if err := VerifyNilBeaconBlock(signed); err != nil {
		return [32]byte{}, err
	}

	// Copy state to avoid mutating the state reference.
	state = state.Copy()

	state, err := ProcessSlots(ctx, state, signed.Block.Slot)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process slot")
	}

	// Execute per block transition.
	state, err = ProcessBlockForStateRoot(ctx, state, signed)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not process block")
	}

	return state.HashTreeRoot(ctx)
}

// ProcessBlockForStateRoot processes the state for state root computation. It skips proposer signature
// and randao signature verifications.
func ProcessBlockForStateRoot(
	ctx context.Context,
	state iface.BeaconState,
	signed *ethpb.SignedBeaconBlockAltair,
) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessBlockForStateRoot")
	defer span.End()
	if err := VerifyNilBeaconBlock(signed); err != nil {
		return nil, err
	}

	blk := signed.Block
	body := blk.Body
	bodyRoot, err := body.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	state, err = b.ProcessBlockHeaderNoVerify(state, blk.Slot, blk.ProposerIndex, blk.ParentRoot, bodyRoot[:])
	if err != nil {
		return nil, errors.Wrap(err, "could not process block header")
	}

	state, err = b.ProcessRandaoNoVerify(state, signed.Block.Body.RandaoReveal)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify and process randao")
	}

	state, err = b.ProcessEth1DataInBlock(ctx, state, signed.Block.Body.Eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not process eth1 data")
	}

	state, err = ProcessOperationsNoVerifyAttsSigs(ctx, state, signed)
	if err != nil {
		return nil, errors.Wrap(err, "could not process block operation")
	}

	return state, nil
}

// ProcessEpoch describes the per epoch operations that are performed on the beacon state.
// It's optimized by pre computing validator attested info and epoch total/attested balances upfront.
//
// Spec code:
// def process_epoch(state: BeaconState) -> None:
//    process_justification_and_finalization(state)  # [Modified in Altair]
//    process_inactivity_updates(state)  # [New in Altair]
//    process_rewards_and_penalties(state)  # [Modified in Altair]
//    process_registry_updates(state)
//    process_slashings(state)  # [Modified in Altair]
//    process_eth1_data_reset(state)
//    process_effective_balance_updates(state)
//    process_slashings_reset(state)
//    process_randao_mixes_reset(state)
//    process_historical_roots_update(state)
//    process_participation_flag_updates(state)  # [New in Altair]
//    process_sync_committee_updates(state)  # [New in Altair]
func ProcessEpoch(ctx context.Context, state iface.BeaconStateAltair) (iface.BeaconStateAltair, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessEpoch")
	defer span.End()

	if state == nil {
		return nil, errors.New("nil state")
	}
	vp, bp, err := InitializeEpochValidators(ctx, state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	vp, bp, err = ProcessEpochParticipation(ctx, state, bp, vp)
	if err != nil {
		return nil, err
	}

	state, err = precompute.ProcessJustificationAndFinalizationPreCompute(state, bp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process justification")
	}

	// New in Altair.
	// process_inactivity_updates is embedded in the below.
	state, err = ProcessRewardsAndPenaltiesPrecompute(state, bp, vp)
	if err != nil {
		return nil, errors.Wrap(err, "could not process rewards and penalties")
	}

	state, err = e.ProcessRegistryUpdates(state)
	if err != nil {
		return nil, errors.Wrap(err, "could not process registry updates")
	}

	// Modified in Altair.
	state, err = ProcessSlashings(state)
	if err != nil {
		return nil, err
	}

	state, err = e.ProcessEth1DataReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessEffectiveBalanceUpdates(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessSlashingsReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessRandaoMixesReset(state)
	if err != nil {
		return nil, err
	}
	state, err = e.ProcessHistoricalRootsUpdate(state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	state, err = ProcessParticipationFlagUpdates(state)
	if err != nil {
		return nil, err
	}

	// New in Altair.
	state, err = ProcessSyncCommitteeUpdates(state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

// ProcessSlots process through skip slots and apply epoch transition when it's needed.
//
// Spec pseudocode definition:
//  def process_slots(state: BeaconState, slot: Slot) -> None:
//    assert state.slot < slot
//    while state.slot < slot:
//        process_slot(state)
//        # Process epoch on the start slot of the next epoch
//        if (state.slot + 1) % SLOTS_PER_EPOCH == 0:
//            process_epoch(state)
//        state.slot = Slot(state.slot + 1)
func ProcessSlots(ctx context.Context, state iface.BeaconState, slot types.Slot) (iface.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "altair.ProcessSlots")
	defer span.End()
	if state == nil {
		return nil, errors.New("nil state")
	}
	span.AddAttributes(trace.Int64Attribute("slots", int64(slot)-int64(state.Slot())))

	// The block must have a higher slot than parent state.
	if state.Slot() >= slot {
		err := fmt.Errorf("expected state.slot %d < slot %d", state.Slot(), slot)
		return nil, err
	}

	highestSlot := state.Slot()
	key, err := coreState.CacheKey(ctx, state)
	if err != nil {
		return nil, err
	}

	// Restart from cached value, if one exists.
	cachedState, err := SkipSlotCache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if cachedState != nil && cachedState.Slot() < slot {
		highestSlot = cachedState.Slot()
		state = cachedState
	}
	if err := SkipSlotCache.MarkInProgress(key); errors.Is(err, cache.ErrAlreadyInProgress) {
		cachedState, err = SkipSlotCache.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		if cachedState != nil && cachedState.Slot() < slot {
			highestSlot = cachedState.Slot()
			state = cachedState
		}
	} else if err != nil {
		return nil, err
	}
	defer func() {
		if err := SkipSlotCache.MarkNotInProgress(key); err != nil {
			log.WithError(err).Error("Failed to mark skip slot no longer in progress")
		}
	}()

	for state.Slot() < slot {
		if ctx.Err() != nil {
			// Cache last best value.
			if highestSlot < state.Slot() {
				if err := SkipSlotCache.Put(ctx, key, state); err != nil {
					log.WithError(err).Error("Failed to put skip slot cache value")
				}
			}
			return nil, ctx.Err()
		}
		state, err = coreState.ProcessSlot(ctx, state)
		if err != nil {
			return nil, errors.Wrap(err, "could not process slot")
		}
		if coreState.CanProcessEpoch(state) {
			state, err = ProcessEpoch(ctx, state)
			if err != nil {
				return nil, errors.Wrap(err, "could not process epoch with optimizations")
			}
		}
		if err := state.SetSlot(state.Slot() + 1); err != nil {
			return nil, errors.Wrap(err, "failed to increment state slot")
		}
	}

	if highestSlot < state.Slot() {
		if err := SkipSlotCache.Put(ctx, key, state); err != nil {
			log.WithError(err).Error("Failed to put skip slot cache value")
		}
	}

	return state, nil
}
