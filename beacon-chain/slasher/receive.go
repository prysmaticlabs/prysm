package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/sirupsen/logrus"
)

// Receive indexed attestations from some source event feed,
// validating their integrity before appending them to an attestation queue
// for batch processing in a separate routine.
func (s *Service) receiveAttestations(ctx context.Context) {
	sub := s.serviceCfg.IndexedAttsFeed.Subscribe(s.indexedAttsChan)
	defer close(s.indexedAttsChan)
	defer sub.Unsubscribe()
	for {
		select {
		case att := <-s.indexedAttsChan:
			attWrapper := &slashertypes.IndexedAttestationWrapper{
				IndexedAttestation: att,
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
	sub := s.serviceCfg.BeaconBlocksFeed.Subscribe(s.beaconBlocksChan)
	defer close(s.beaconBlocksChan)
	defer sub.Unsubscribe()
	for {
		select {
		case blockHeader := <-s.beaconBlocksChan:
			// TODO(#8331): Defer blocks from the future for later processing.
			wrappedProposal := &slashertypes.SignedBlockHeaderWrapper{
				SignedBeaconBlockHeader: blockHeader,
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
func (s *Service) processQueuedAttestations(ctx context.Context, epochTicker <-chan types.Epoch) {
	for {
		select {
		case currentEpoch := <-epochTicker:
			// We take all the attestations in the queue and filter out
			// those which are valid now and valid in the future.
			attestations := s.attsQueue.dequeue()
			validAtts, validInFuture, numDropped := s.validateAttestationIntegrity(attestations, currentEpoch)

			deferredAttestationsTotal.Add(float64(len(validInFuture)))
			droppedAttestationsTotal.Add(float64(numDropped))

			// We add back those attestations that are valid in the future to the queue.
			s.attsQueue.extend(validInFuture)

			log.WithFields(logrus.Fields{
				"currentEpoch":    currentEpoch,
				"numValidAtts":    len(validAtts),
				"numDeferredAtts": len(validInFuture),
				"numDroppedAtts":  numDropped,
			}).Info("Epoch reached, processing queued atts for slashing detection")

			// Save the attestation records to our database.
			if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(
				ctx, validAtts,
			); err != nil {
				log.WithError(err).Error("Could not save attestation records to DB")
				continue
			}

			groupedAtts := s.groupByValidatorChunkIndex(validAtts)
			// TODO(#8331): Consider using goroutines and wait groups here.
			for validatorChunkIdx, batch := range groupedAtts {
				if err := s.detectSlashableAttestations(ctx, &chunkUpdateArgs{
					validatorChunkIndex: validatorChunkIdx,
					currentEpoch:        currentEpoch,
				}, batch); err != nil {
					log.WithError(err).Error("Could not detect slashable attestations")
					continue
				}
			}

			processedAttestationsTotal.Add(float64(len(validAtts)))
		case <-ctx.Done():
			return
		}
	}
}

// Process queued blocks every time an epoch ticker fires. We retrieve
// these blocks from a queue, then perform double proposal detection.
func (s *Service) processQueuedBlocks(ctx context.Context, epochTicker <-chan types.Epoch) {
	for {
		select {
		case currentEpoch := <-epochTicker:
			blocks := s.blksQueue.dequeue()

			receivedBlocksTotal.Add(float64(len(blocks)))

			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numBlocks":    len(blocks),
			}).Info("Epoch reached, processing queued blocks for slashing detection")

			if err := s.detectSlashableBlocks(ctx, blocks); err != nil {
				log.WithError(err).Error("Could not detect slashable blocks")
				continue
			}
			processedBlocksTotal.Add(float64(len(blocks)))
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) pruneSlasherData(ctx context.Context, epochTicker <-chan types.Epoch) {
	for {
		select {
		case currentEpoch := <-epochTicker:
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
