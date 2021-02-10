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
			// TODO(#8331): Defer attestations from the future for later processing.
			if !validateAttestationIntegrity(att) {
				continue
			}
			compactAtt := &slashertypes.CompactAttestation{
				AttestingIndices: att.AttestingIndices,
				Source:           att.Data.Source.Epoch,
				Target:           att.Data.Target.Epoch,
			}
			s.queueLock.Lock()
			s.attestationQueue = append(s.attestationQueue, compactAtt)
			s.queueLock.Unlock()
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
func (s *Service) processQueuedAttestations(ctx context.Context, epochTicker <-chan uint64) {
	for {
		select {
		case currentEpoch := <-epochTicker:
			s.queueLock.Lock()
			attestations := s.attestationQueue
			s.attestationQueue = make([]*slashertypes.CompactAttestation, 0)
			s.queueLock.Unlock()
			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numAtts":      len(attestations),
			}).Info("Epoch reached, processing queued atts for slashing detection")
			groupedAtts := s.groupByValidatorChunkIndex(attestations)
			for validatorChunkIdx, batch := range groupedAtts {
				validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIdx)
				// Save the attestation records for the validator indices in our database.
				if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(
					ctx, validatorIndices, batch,
				); err != nil {
					log.WithError(err).Error("Could not save attestation records to DB")
					continue
				}
				if err := s.detectSlashableAttestations(ctx, &chunkUpdateOptions{
					validatorChunkIndex: validatorChunkIdx,
					currentEpoch:        types.Epoch(currentEpoch),
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
