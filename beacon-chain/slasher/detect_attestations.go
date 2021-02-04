package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/sirupsen/logrus"
)

// Process queued attestations every time an epoch ticker fires. We retrieve
// these attestations from a queue, then group them all by validator chunk index.
// This grouping will allow us to perform detection on batches of attestations
// per validator chunk index which can be done concurrently.
func (s *Service) processQueuedAttestations(ctx context.Context, epochTicker <-chan uint64) {
	for {
		select {
		case currentEpoch := <-epochTicker:
			s.queueLock.Lock()
			atts := s.attestationQueue
			s.attestationQueue = make([]*CompactAttestation, 0)
			s.queueLock.Unlock()
			log.WithFields(logrus.Fields{
				"currentEpoch": currentEpoch,
				"numAtts":      len(atts),
			}).Info("Epoch reached, processing queued atts for slashing detection")
			groupedAtts := s.groupByValidatorChunkIndex(atts)
			for validatorChunkIdx, attsBatch := range groupedAtts {
				validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIdx)
				// Save the attestation records for the validator indices in our database.
				if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(
					ctx, validatorIndices, attsBatch,
				); err != nil {
					panic(err)
				}
				// Detect slashings within a batch of attestations for a validator chunk index.
				s.detectAttestationBatch(attsBatch, validatorChunkIdx, types.Epoch(currentEpoch))
			}
		case <-ctx.Done():
			return
		}
	}
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection over the batch.
// The process is as follows given a batch of attestations:
//
// 1. Categorize the attestations by chunk index.
// 2. For every validator in a validator chunk index, update all the chunks that need to be
//    updated based on the current epoch, and return these updated chunks.
// 3. Using the chunks from step (2), for every attestation by chunk index, for each
//    validator in its attesting indices:
//    - Check if the attestation is slashable, if so return a slashing object
//    - Update all min chunks
//    - Update all max chunks
// 4. Update the latest written epoch for all validators involved to the current epoch.
//
// This function performs a lot of critical actions and is split into smaller helpers for cleanliness.
func (s *Service) detectAttestationBatch(
	attBatch []*CompactAttestation, validatorChunkIndex uint64, currentEpoch types.Epoch,
) {
	// We categorize attestations by chunk index.
	attestationsByChunk := s.groupByChunkIndex(attBatch)

	updatedChunks := s.updateChunksForCurrentEpoch()

	slashings := s.updateMaxChunks()
	moreSlashings := s.updateMinChunks()

	totalSlashings := append(slashings, moreSlashings...)
	for _, sl := range totalSlashings {
		if sl != notSlashable {
			log.Infof("Slashing found: %s", sl)
		}
	}

	// Update all relevant validators for current epoch.
	validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIndex)
	if err := s.slasherDB.UpdateLatestEpochWrittenForValidators(ctx, validatorIndices, currentEpoch); err != nil {
		panic(err)
	}
}

func (s *Service) updateMaxChunks() []byte {
	return nil
}
func (s *Service) updateMinChunks() []byte {
	return nil
}

func (s *Service) updateChunksForCurrentEpoch() int {}

func (s *Service) groupByChunkIndex(attestations []*CompactAttestation) map[uint64][]*CompactAttestation {
	attestationsByChunkIndex := make(map[uint64][]*CompactAttestation)
	for _, att := range attestations {
		chunkIdx := s.params.chunkIndex(types.Epoch(att.Source))
		attestationsByChunkIndex[chunkIdx] = append(attestationsByChunkIndex[chunkIdx], att)
	}
	return attestationsByChunkIndex
}

// Group a list of attestations into batches by validator chunk index.
// This way, we can detect on the batch of attestations for each validator chunk index
// concurrently, and also allowing us to effectively use a single 2D chunk
// for slashing detection through this logical grouping.
func (s *Service) groupByValidatorChunkIndex(
	attestations []*CompactAttestation,
) map[uint64][]*CompactAttestation {
	groupedAttestations := make(map[uint64][]*CompactAttestation)
	for _, att := range attestations {
		validatorChunkIndices := make(map[uint64]bool)
		for _, validatorIdx := range att.AttestingIndices {
			validatorChunkIndex := s.params.validatorChunkIndex(types.ValidatorIndex(validatorIdx))
			validatorChunkIndices[validatorChunkIndex] = true
		}
		for validatorChunkIndex := range validatorChunkIndices {
			groupedAttestations[validatorChunkIndex] = append(
				groupedAttestations[validatorChunkIndex],
				att,
			)
		}
	}
	return groupedAttestations
}
