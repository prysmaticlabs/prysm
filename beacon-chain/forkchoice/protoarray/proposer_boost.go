package protoarray

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// BoostProposerRoot sets the block root which should be boosted during
// the LMD fork choice algorithm calculations. This is meant to reward timely,
// proposed blocks which occur before a cutoff interval set to
// SECONDS_PER_SLOT // INTERVALS_PER_SLOT.
//
//  time_into_slot = (store.time - store.genesis_time) % SECONDS_PER_SLOT
//  is_before_attesting_interval = time_into_slot < SECONDS_PER_SLOT // INTERVALS_PER_SLOT
//  if get_current_slot(store) == block.slot and is_before_attesting_interval:
//      store.proposer_boost_root = hash_tree_root(block)
func (f *ForkChoice) BoostProposerRoot(ctx context.Context, blockSlot types.Slot, blockRoot [32]byte, genesisTime time.Time) error {
	secondsPerSlot := params.BeaconConfig().SecondsPerSlot
	timeIntoSlot := uint64(time.Since(genesisTime).Seconds()) % secondsPerSlot
	isBeforeAttestingInterval := timeIntoSlot < secondsPerSlot/params.BeaconConfig().IntervalsPerSlot
	currentSlot := slots.SinceGenesis(genesisTime)

	// Only update the boosted proposer root to the incoming block root.
	// If the block is for the current, clock-based slot and the block was timely.
	if currentSlot == blockSlot && isBeforeAttestingInterval {
		f.store.proposerBoostLock.Lock()
		f.store.proposerBoostRoot = blockRoot
		f.store.proposerBoostLock.Unlock()
	}
	return nil
}
