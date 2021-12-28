package light

import (
	"bytes"
	"context"

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

func (s *Service) subscribeEvents(ctx context.Context) {
	c := make(chan *feed.Event, 1)
	sub := s.cfg.StateNotifier.StateFeed().Subscribe(c)
	defer sub.Unsubscribe()
	slotTicker := slots.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case slot := <-slotTicker.C():
			slotsPerPeriod := params.BeaconConfig().SlotsPerEpoch * types.Slot(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
			if slot%slotsPerPeriod == 1 {
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

	log.WithFields(logrus.Fields{
		"startSlot": startSlot,
		"endSlot":   endSlot,
		"count":     len(updates),
	}).Info("Pruning updates")

	var hasFinalized bool
	var hasBestNonFinalized bool
	var bestFinalizedUpdateSlot types.Slot
	var bestFinalizedUpdateBitCount uint64
	var bestNonFinalizedUpdateSlot types.Slot
	var bestNonFinalizedUpdateBitCount uint64
	for i := len(updates) - 1; i >= 0; i-- {
		u := updates[i]
		log.WithFields(logrus.Fields{
			"slot":               u.AttestedHeader.Slot,
			"finalityHeaderSlot": u.FinalityHeader.Slot,
			"AggBitCount":        u.SyncAggregate.SyncCommitteeBits.Count(),
		}).Info("Pruning update")

		if !emptyHeader(u.FinalityHeader) {
			hasFinalized = true
			if u.SyncAggregate.SyncCommitteeBits.Count() > bestFinalizedUpdateBitCount {
				bestFinalizedUpdateSlot = u.AttestedHeader.Slot
				bestFinalizedUpdateBitCount = u.SyncAggregate.SyncCommitteeBits.Count()
				if bestFinalizedUpdateBitCount == params.BeaconConfig().SyncCommitteeSize {
					log.Info("Found best finalized update")
					break
				}
			}
			continue
		}

		if !hasFinalized && !hasBestNonFinalized {
			if u.SyncAggregate.SyncCommitteeBits.Count() > bestNonFinalizedUpdateBitCount {
				bestNonFinalizedUpdateSlot = u.AttestedHeader.Slot
				bestNonFinalizedUpdateBitCount = u.SyncAggregate.SyncCommitteeBits.Count()
				if bestNonFinalizedUpdateBitCount == params.BeaconConfig().SyncCommitteeSize {
					log.Info("Found best non finalized update")
					hasBestNonFinalized = true
				}
			}
		}
	}

	if bestNonFinalizedUpdateSlot == 0 && bestFinalizedUpdateSlot == 0 {
		return nil
	}

	log.WithFields(logrus.Fields{
		"hasFinalized":                   hasFinalized,
		"hasBestNonFinalized":            hasBestNonFinalized,
		"bestFinalizedUpdateSlot":        bestFinalizedUpdateSlot,
		"bestFinalizedUpdateBitCount":    bestFinalizedUpdateBitCount,
		"bestNonFinalizedUpdateSlot":     bestNonFinalizedUpdateSlot,
		"bestNonFinalizedUpdateBitCount": bestNonFinalizedUpdateBitCount,
	}).Info("Deleting updates from db")

	if hasFinalized {
		slots := make([]types.Slot, endSlot-startSlot+1)
		i := 0
		for slot := startSlot; slot <= endSlot; slot++ {
			slots[i] = slot
			i++
		}
		if err := s.cfg.BeaconDB.DeleteLightClientUpdates(ctx, slots); err != nil {
			return err
		}

		slots = make([]types.Slot, bestFinalizedUpdateSlot-startSlot)
		i = 0
		for slot := startSlot; slot < bestFinalizedUpdateSlot; slot++ {
			slots[i] = slot
			i++
		}
		if err := s.cfg.BeaconDB.DeleteLightClientFinalizedUpdates(ctx, slots); err != nil {
			return err
		}

		slots = make([]types.Slot, endSlot-bestFinalizedUpdateSlot)
		i = 0
		for slot := bestFinalizedUpdateSlot + 1; slot <= endSlot; slot++ {
			slots[i] = slot
			i++
		}
		if err := s.cfg.BeaconDB.DeleteLightClientFinalizedUpdates(ctx, slots); err != nil {
			return err
		}
	} else {
		slots := make([]types.Slot, bestNonFinalizedUpdateSlot-startSlot)
		i := 0
		for slot := startSlot; slot < bestNonFinalizedUpdateSlot; slot++ {
			slots[i] = slot
			i++
		}
		if err := s.cfg.BeaconDB.DeleteLightClientUpdates(ctx, slots); err != nil {
			return err
		}

		slots = make([]types.Slot, endSlot-bestNonFinalizedUpdateSlot)
		i = 0
		for slot := bestNonFinalizedUpdateSlot + 1; slot <= endSlot; slot++ {
			slots[i] = slot
			i++
		}
		if err := s.cfg.BeaconDB.DeleteLightClientUpdates(ctx, slots); err != nil {
			return err
		}
	}
	return nil
}

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
	finalityHeader := &ethpb.BeaconBlockHeader{
		ParentRoot: make([]byte, fieldparams.RootLength),
		StateRoot:  make([]byte, fieldparams.RootLength),
		BodyRoot:   make([]byte, fieldparams.RootLength),
	}
	update := &ethpb.LightClientUpdate{
		AttestedHeader:          h.Header,
		FinalityHeader:          finalityHeader,
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
	return s.cfg.BeaconDB.SaveFinalizedLightClientUpdate(ctx, update)
}

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
