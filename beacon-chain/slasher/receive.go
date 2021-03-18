package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/sirupsen/logrus"
)

// Receive indexed attestations from some source event feed,
// validating their integrity before appending them to an attestation queue
// for batch processing in a separate routine.
func (s *Service) receiveAttestations(ctx context.Context) {
	sub := s.serviceCfg.IndexedAttestationsFeed.Subscribe(s.indexedAttsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case att := <-s.indexedAttsChan:
			// TODO(#8331): Defer attestations from the future for later processing.
			if !validateAttestationIntegrity(att) {
				continue
			}
			signingRoot, err := att.Data.HashTreeRoot()
			if err != nil {
				log.WithError(err).Error("Could not get hash tree root of attestation")
				continue
			}
			attWrapper := &slashertypes.IndexedAttestationWrapper{
				IndexedAttestation: att,
				SigningRoot:        signingRoot,
			}
			s.attsQueue.push(attWrapper)
		case err := <-sub.Err():
			log.WithError(err).Debug("Subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}

// Receive beacon blocks from some source event feed,
func (s *Service) receiveBlocks(ctx context.Context) {
	sub := s.serviceCfg.BeaconBlockHeadersFeed.Subscribe(s.beaconBlockHeadersChan)
	defer close(s.beaconBlockHeadersChan)
	defer sub.Unsubscribe()
	for {
		select {
		case blockHeader := <-s.beaconBlockHeadersChan:
			// TODO(#8331): Defer blocks from the future for later processing.
			signingRoot, err := blockHeader.Header.HashTreeRoot()
			if err != nil {
				log.WithError(err).Error("Could not get hash tree root of signed block header")
				continue
			}
			wrappedProposal := &slashertypes.SignedBlockHeaderWrapper{
				SignedBeaconBlockHeader: blockHeader,
				SigningRoot:             signingRoot,
			}
			s.blksQueue.push(wrappedProposal)
		case err := <-sub.Err():
			log.WithError(err).Debug("Subscriber closed with error")
			return
		case <-ctx.Done():
			return
		}
	}
}

// Process queued attestations every time an epoch ticker fires. We retrieve
// these attestations from a queue, then group them all by validator chunk index.
// This grouping will allow us to perform detection on batches of attestations
// per validator chunk index which can be done concurrently.
func (s *Service) processQueuedAttestations(ctx context.Context, slotTicker <-chan types.Slot) {
	for {
		select {
		case currentSlot := <-slotTicker:
			attestations := s.attsQueue.dequeue()
			currentEpoch := helpers.SlotToEpoch(currentSlot)
			// We take all the attestations in the queue and filter out
			// those which are valid now and valid in the future.
			validAtts, validInFuture, numDropped := s.filterAttestations(attestations, currentEpoch)

			deferredAttestationsTotal.Add(float64(len(validInFuture)))
			droppedAttestationsTotal.Add(float64(numDropped))

			// We add back those attestations that are valid in the future to the queue.
			s.attsQueue.extend(validInFuture)

			log.WithFields(logrus.Fields{
				"currentSlot":     currentSlot,
				"currentEpoch":    currentEpoch,
				"numValidAtts":    len(validAtts),
				"numDeferredAtts": len(validInFuture),
				"numDroppedAtts":  numDropped,
			}).Info("New slot, processing queued atts for slashing detection")

			// Save the attestation records to our database.
			if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(
				ctx, validAtts,
			); err != nil {
				log.WithError(err).Error("Could not save attestation records to DB")
				continue
			}

			// Check for slashings.
			_, err := s.CheckSlashableAttestations(ctx, validAtts)
			if err != nil {
				log.WithError(err).Error("Could not check slashable attestations")
				continue
			}

			processedAttestationsTotal.Add(float64(len(validAtts)))
		case <-ctx.Done():
			return
		}
	}
}

// Process queued blocks every time an epoch ticker fires. We retrieve
// these blocks from a queue, then perform double proposal detection.
func (s *Service) processQueuedBlocks(ctx context.Context, slotTicker <-chan types.Slot) {
	for {
		select {
		case currentSlot := <-slotTicker:
			blocks := s.blksQueue.dequeue()
			currentEpoch := helpers.SlotToEpoch(currentSlot)

			receivedBlocksTotal.Add(float64(len(blocks)))

			log.WithFields(logrus.Fields{
				"currentSlot":  currentSlot,
				"currentEpoch": currentEpoch,
				"numBlocks":    len(blocks),
			}).Info("New slot, processing queued blocks for slashing detection")

			proposerSlashings, err := s.detectProposerSlashings(ctx, blocks)
			if err != nil {
				log.WithError(err).Error("Could not detect slashable blocks")
				continue
			}

			s.recordDoubleProposals(proposerSlashings)

			processedBlocksTotal.Add(float64(len(blocks)))
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) pruneSlasherData(ctx context.Context, slotTicker <-chan types.Slot) {
	for {
		select {
		case currentSlot := <-slotTicker:
			if !helpers.IsEpochStart(currentSlot) {
				continue
			}
			currentEpoch := helpers.SlotToEpoch(currentSlot)
			if err := s.serviceCfg.Database.PruneAttestations(ctx, currentEpoch, s.params.historyLength); err != nil {
				log.WithError(err).Error("Could not prune attestations")
				continue
			}

			if err := s.serviceCfg.Database.PruneProposals(ctx, currentEpoch, s.params.historyLength); err != nil {
				log.WithError(err).Error("Could not prune proposals")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
