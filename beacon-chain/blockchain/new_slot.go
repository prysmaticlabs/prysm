package blockchain

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/time/slots"
)

// NewSlot mimics the implementation of `on_tick` in fork choice consensus spec.
// It resets the proposer boost root in fork choice, and it updates store's justified checkpoint
// if a better checkpoint on the store's finalized checkpoint chain.
// This should only be called at the start of every slot interval.
//
// Spec pseudocode definition:
//    # Reset store.proposer_boost_root if this is a new slot
//    if current_slot > previous_slot:
//        store.proposer_boost_root = Root()
//
//    # Not a new epoch, return
//    if not (current_slot > previous_slot and compute_slots_since_epoch_start(current_slot) == 0):
//        return
//
//    # Update store.justified_checkpoint if a better checkpoint on the store.finalized_checkpoint chain
//    if store.best_justified_checkpoint.epoch > store.justified_checkpoint.epoch:
//        finalized_slot = compute_start_slot_at_epoch(store.finalized_checkpoint.epoch)
//        ancestor_at_finalized_slot = get_ancestor(store, store.best_justified_checkpoint.root, finalized_slot)
//        if ancestor_at_finalized_slot == store.finalized_checkpoint.root:
//            store.justified_checkpoint = store.best_justified_checkpoint
func (s *Service) NewSlot(ctx context.Context, slot types.Slot) error {

	// Reset proposer boost root in fork choice.
	if err := s.cfg.ForkChoiceStore.ResetBoostedProposerRoot(ctx); err != nil {
		return errors.Wrap(err, "could not reset boosted proposer root in fork choice")
	}

	// Return if it's not a new epoch.
	if !slots.IsEpochStart(slot) {
		return nil
	}

	// Update store.justified_checkpoint if a better checkpoint on the store.finalized_checkpoint chain
	bj, err := s.store.BestJustifiedCheckpt()
	if err != nil {
		return errors.Wrap(err, "could not get best justified checkpoint")
	}
	j, err := s.store.JustifiedCheckpt()
	if err != nil {
		return errors.Wrap(err, "could not get justified checkpoint")
	}
	f, err := s.store.FinalizedCheckpt()
	if err != nil {
		return errors.Wrap(err, "could not get finalized checkpoint")
	}
	if bj.Epoch > j.Epoch {
		finalizedSlot, err := slots.EpochStart(f.Epoch)
		if err != nil {
			return err
		}
		r, err := s.ancestor(ctx, bj.Root, finalizedSlot)
		if err != nil {
			return err
		}
		if bytes.Equal(r, f.Root) {
			h, err := s.getPayloadHash(ctx, bj.Root)
			if err != nil {
				return err
			}
			s.store.SetJustifiedCheckptAndPayloadHash(bj, h)
		}
	}
	return nil

}
