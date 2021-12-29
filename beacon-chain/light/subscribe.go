package light

import (
	"bytes"
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/block"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
)

// subscribeEvents subscribe to new block processed and new finalized events.
// Based on the event, it will call the corresponding handler.
func (s *Service) subscribeEvents(ctx context.Context) {
	c := make(chan *feed.Event, 1)
	sub := s.cfg.StateNotifier.StateFeed().Subscribe(c)
	defer sub.Unsubscribe()
	slotTicker := slots.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case slot := <-slotTicker.C():
			if firstSlotSyncCommittee(slot) {
				if err := s.pruneUpdates(ctx, slots.PrevSlot(slot)); err != nil {
					log.WithError(err).Error("Could not prune updates")
					continue
				}
			}
		case ev := <-c:
			if ev.Type == statefeed.BlockProcessed {
				d, ok := ev.Data.(*statefeed.BlockProcessedData)
				if !ok {
					continue
				}
				if d.SignedBlock == nil || d.PostState == nil {
					continue
				}
				d.SignedBlock.Block().Slot()
				if err := s.newBlock(ctx, d.BlockRoot, d.SignedBlock, d.PostState); err != nil {
					log.WithError(err).Error("Could not process new block")
					continue
				}
			} else if ev.Type == statefeed.FinalizedCheckpoint {
				d, ok := ev.Data.(*statefeed.NewFinalizedData)
				if !ok {
					continue
				}
				if d.SignedBlock == nil || d.PostState == nil {
					continue
				}
				if err := s.newFinalized(ctx, d.SignedBlock, d.PostState); err != nil {
					log.WithError(err).Error("Could not process new finalized")
					continue
				}
			}
		case err := <-sub.Err():
			log.WithError(err).Error("Could not subscribe to state notifier")
			return
		case <-ctx.Done():
			log.Debug("Context closed, exiting goroutine")
			return
		}
	}
}

// pruneUpdates is a handler to prune light client updates that are older than the current sync committee period.
func (s *Service) pruneUpdates(ctx context.Context, startSlot types.Slot) error {
	slotsPerSyncPeriod := params.BeaconConfig().SlotsPerEpoch * types.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	if startSlot < slotsPerSyncPeriod {
		return nil
	}
	startSlot -= slotsPerSyncPeriod
	endSlot := startSlot + slotsPerSyncPeriod - 1
	f := filters.NewFilter().SetStartSlot(startSlot).SetEndSlot(endSlot)
	updates, err := s.cfg.BeaconDB.LightClientUpdates(ctx, f)
	if err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}

	log.WithFields(logrus.Fields{
		"updateStartSlot": startSlot,
		"updateEndSlot":   endSlot,
		"updateCount":     len(updates),
	}).Info("Pruning updates")

	var hasFinalized, hasBestNonFinalized bool
	var bestFinalizedUpdateSlot, bestNonFinalizedUpdateSlot types.Slot
	var bestFinalizedUpdateBitCount, bestNonFinalizedUpdateBitCount uint64

	// Starting at highest slot to find the best update. Higher the slot the better.
	for i := len(updates) - 1; i >= 0; i-- {
		u := updates[i]
		if !emptyHeader(u.FinalityHeader) {
			hasFinalized = true
			if u.SyncAggregate.SyncCommitteeBits.Count() > bestFinalizedUpdateBitCount {
				bestFinalizedUpdateSlot = u.AttestedHeader.Slot
				bestFinalizedUpdateBitCount = u.SyncAggregate.SyncCommitteeBits.Count()
				// Once we find the best finalized update that reaches max voting quorum, we can stop looking.
				if bestFinalizedUpdateBitCount == params.BeaconConfig().SyncCommitteeSize {
					break
				}
			}
			continue
		}
		// Only check for non finalized updates if we haven't found a finalized update.
		if !hasFinalized && !hasBestNonFinalized {
			if u.SyncAggregate.SyncCommitteeBits.Count() > bestNonFinalizedUpdateBitCount {
				bestNonFinalizedUpdateSlot = u.AttestedHeader.Slot
				bestNonFinalizedUpdateBitCount = u.SyncAggregate.SyncCommitteeBits.Count()
				// Once we find the best non finalized update that reaches max voting quorum, we can stop looking.
				if bestNonFinalizedUpdateBitCount == params.BeaconConfig().SyncCommitteeSize {
					hasBestNonFinalized = true
				}
			}
		}
	}

	if hasFinalized {
		if err := deleteUpdates(ctx, startSlot, endSlot+1, s.cfg.BeaconDB.DeleteLightClientUpdates); err != nil {
			return err
		}
		if err := deleteUpdates(ctx, startSlot, bestFinalizedUpdateSlot, s.cfg.BeaconDB.DeleteLightClientFinalizedUpdates); err != nil {
			return err
		}
		if err := deleteUpdates(ctx, bestFinalizedUpdateSlot+1, endSlot+1, s.cfg.BeaconDB.DeleteLightClientFinalizedUpdates); err != nil {
			return err
		}
	} else {
		if err := deleteUpdates(ctx, startSlot, bestNonFinalizedUpdateSlot, s.cfg.BeaconDB.DeleteLightClientUpdates); err != nil {
			return err
		}
		if err := deleteUpdates(ctx, bestFinalizedUpdateSlot+1, endSlot+1, s.cfg.BeaconDB.DeleteLightClientUpdates); err != nil {
			return err
		}
	}
	return nil
}

// newBlock updates attested header for a light client update and saves it to the database.
func (s *Service) newBlock(ctx context.Context, r [32]byte, blk block.SignedBeaconBlock, st state.BeaconState) error {
	if st.Version() == version.Phase0 || blk.Version() == version.Phase0 {
		return nil
	}
	h, err := blk.Header()
	if err != nil {
		return err
	}
	com, err := st.NextSyncCommittee()
	if err != nil {
		return err
	}
	b, err := st.NextSyncCommitteeProof()
	if err != nil {
		return err
	}

	update := &ethpb.LightClientUpdate{
		AttestedHeader: h.Header,
		FinalityHeader: &ethpb.BeaconBlockHeader{
			ParentRoot: make([]byte, fieldparams.RootLength),
			StateRoot:  make([]byte, fieldparams.RootLength),
			BodyRoot:   make([]byte, fieldparams.RootLength),
		},
		NextSyncCommittee:       com,
		NextSyncCommitteeBranch: b,
		ForkVersion:             st.Fork().CurrentVersion,
	}

	if err := s.saveUpdate(r[:], update); err != nil {
		return err
	}

	parentRoot := blk.Block().ParentRoot()
	update, err = s.getUpdate(parentRoot)
	if err != nil {
		return err
	}
	if update == nil {
		return nil
	}
	agg, err := blk.Block().Body().SyncAggregate()
	if err != nil {
		return err
	}
	update.SyncAggregate = agg
	return s.cfg.BeaconDB.SaveLightClientUpdate(ctx, update)
}

// newFinalized updates finalized header for a light client update and saves it to the database.
func (s *Service) newFinalized(ctx context.Context, blk block.SignedBeaconBlock, st state.BeaconState) error {
	if blk.Version() == version.Phase0 {
		return nil
	}
	parentRoot := blk.Block().ParentRoot()
	update, err := s.getUpdate(parentRoot)
	if err != nil {
		return err
	}
	if update == nil {
		return nil
	}

	fb, err := s.cfg.BeaconDB.Block(ctx, bytesutil.ToBytes32(st.FinalizedCheckpoint().Root))
	if err != nil {
		return err
	}
	fh, err := fb.Header()
	if err != nil {
		return err
	}
	update.FinalityHeader = fh.Header

	fp, err := st.FinalizedRootProof()
	if err != nil {
		return err
	}
	update.FinalityBranch = fp
	if err := s.cfg.BeaconDB.DeleteLightClientUpdates(ctx, []types.Slot{update.AttestedHeader.Slot}); err != nil {
		return err
	}
	return s.cfg.BeaconDB.SaveFinalizedLightClientUpdate(ctx, update)
}

// emptyHeader returns true if the header is empty.
func emptyHeader(header *ethpb.BeaconBlockHeader) bool {
	if header.Slot != 0 {
		return false
	}
	if header.ProposerIndex != 0 {
		return false
	}
	if !bytes.Equal(header.ParentRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(header.StateRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	if !bytes.Equal(header.BodyRoot, make([]byte, fieldparams.RootLength)) {
		return false
	}
	return true
}

// firstSlotSyncCommittee returns true if input slot is the first slot of a sync committee period.
func firstSlotSyncCommittee(slot types.Slot) bool {
	slotsPerPeriod := params.BeaconConfig().SlotsPerEpoch * types.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	return slot%slotsPerPeriod == 1
}

// deleteUpdates deletes light client updates from the database using start slot and end slot as range.
func deleteUpdates(ctx context.Context, startSlot, endSlot types.Slot, deleteFunc func(ctx context.Context, slots []types.Slot) error) error {
	if startSlot > endSlot {
		return fmt.Errorf("start slot %d is greater than end slot %d", startSlot, endSlot)
	}
	s := make([]types.Slot, endSlot-startSlot)
	for i := 0; i < len(s); i++ {
		s[i] = startSlot + types.Slot(i)
	}
	return deleteFunc(ctx, s)
}
