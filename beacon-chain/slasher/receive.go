package slasher

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
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
			// TODO(#8331): Defer attestations from the future for later processing.
			if !validateAttestationIntegrity(att) {
				continue
			}
			signingRoot, err := att.Data.HashTreeRoot()
			if err != nil {
				log.WithError(err).Debug("Subscriber closed with error")
				return
			}
			compactAtt := &slashertypes.CompactAttestation{
				AttestingIndices: att.AttestingIndices,
				Source:           att.Data.Source.Epoch,
				Target:           att.Data.Target.Epoch,
				SigningRoot:      signingRoot,
			}
			s.attestationQueueLock.Lock()
			s.attestationQueue = append(s.attestationQueue, compactAtt)
			s.attestationQueueLock.Unlock()
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
			signingRoot, err := blockHeader.HashTreeRoot()
			if err != nil {
				log.WithError(err).Debug("Subscriber closed with error")
				return
			}
			compactBlock := &slashertypes.CompactBeaconBlock{
				ProposerIndex: types.ValidatorIndex(blockHeader.ProposerIndex),
				Slot:          blockHeader.Slot,
				SigningRoot:   signingRoot,
			}
			s.blockQueueLock.Lock()
			s.beaconBlocksQueue = append(s.beaconBlocksQueue, compactBlock)
			s.blockQueueLock.Unlock()
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
			s.attestationQueueLock.Lock()
			attestations := s.attestationQueue
			fmt.Printf("Queue loaded! %d atts loaded\n", len(attestations))
			s.attestationQueue = make([]*slashertypes.CompactAttestation, 0)
			s.attestationQueueLock.Unlock()
			currentEpoch := helpers.SlotToEpoch(currentSlot)
			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numAtts":      len(attestations),
			}).Info("Epoch reached, processing queued atts for slashing detection")
			// Save the attestation records to our database.
			if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(
				ctx, attestations,
			); err != nil {
				log.WithError(err).Error("Could not save attestation records to DB")
				continue
			}

			groupedAtts := s.groupByValidatorChunkIndex(attestations)
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
			s.blockQueueLock.Lock()
			blocks := s.beaconBlocksQueue
			s.beaconBlocksQueue = make([]*slashertypes.CompactBeaconBlock, 0)
			s.blockQueueLock.Unlock()
			currentEpoch := helpers.SlotToEpoch(currentSlot)
			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numBlocks":    len(blocks),
			}).Info("Epoch reached, processing queued blocks for slashing detection")
			if err := s.detectSlashableBlocks(ctx, blocks); err != nil {
				log.WithError(err).Error("Could not detect slashable blocks")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}
