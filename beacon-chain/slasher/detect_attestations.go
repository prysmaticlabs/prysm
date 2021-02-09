package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
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
			s.attestationQueue = make([]*slashertypes.CompactAttestation, 0)
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
					log.WithError(err).Error("Could not save attestation records to DB")
					continue
				}
				s.detectAttestationBatch(ctx, attsBatch, validatorChunkIdx, types.Epoch(currentEpoch))
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
// 1. Group the attestations by chunk index.
// 2. Update the min and max spans for those grouped attestations, check if any slashings are
//    found in the process
// 3. Update the latest written epoch for all validator indices involved up and
//    including the current epoch
//
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
	ctx context.Context,
	attBatch []*slashertypes.CompactAttestation,
	validatorChunkIndex uint64,
	currentEpoch types.Epoch,
) error {
	// Group attestations by chunk index.
	attestationsByChunkIdx := s.groupByChunkIndex(attBatch)

	_, err := s.updateMinSpans()
	if err != nil {
		return err
	}
	//_, err = s.updateMaxSpans()
	//if err != nil {
	//	return err
	//}

	// Update the latest written epoch for all involved validator indices.
	validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIndex)
	return s.serviceCfg.Database.SaveLatestEpochAttestedForValidators(ctx, validatorIndices, currentEpoch)
}

func (s *Service) updateMinSpans(
	ctx context.Context,
	validatorChunkIdx uint64,
	currentEpoch types.Epoch,
) (slashertypes.SlashingKind, error) {
	// Update the required chunks with the current epoch.
	validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIdx)
	epochs, epochsExit, err := s.serviceCfg.Database.LatestEpochAttestedForValidators(ctx, validatorIndices)
	if err != nil {
		return slashertypes.NotSlashable, nil
	}
	updatedChunks, err := s.updateChunksWithCurrentEpoch()
	if err != nil {
		return slashertypes.NotSlashable, nil
	}

	// Apply the attestations to the related min chunks.
	// TODO: Apply...

	// Write the updated chunks to disk.
	return slashertypes.NotSlashable, nil
}

func (s *Service) updateChunksWithCurrentEpoch(
	ctx context.Context,
	validatorChunkIdx uint64,
	validatorIdx types.ValidatorIndex,
	epoch,
	currentEpoch types.Epoch,
) (updatedChunks map[uint64]Chunker, err error) {
	updatedChunks = make(map[uint64]Chunker)
	updatedChunks = make(map[uint64]Chunker)
	for epoch <= currentEpoch {
		chunkIdx := s.params.chunkIndex(epoch)
		currentChunk := s.chunkForUpdate(updatedChunks, validatorChunkIdx, chunkIdx, slashertypes.MinSpan)
		for s.params.chunkIndex(epoch) == chunkIdx && epoch <= currentEpoch {
			if err = setChunkDataAtEpoch(
				s.params,
				currentChunk.Chunk(),
				validatorIdx,
				epoch,
				types.Epoch(currentChunk.NeutralElement())+epoch,
			); err != nil {
				return
			}
			epoch++
		}
		updatedChunks[chunkIdx] = currentChunk
	}
	return
}

func (s *Service) chunkForUpdate(
	chunksByChunkIndex map[uint64]Chunker,
	validatorChunkIndex,
	chunkIndex uint64,
	kind slashertypes.ChunkKind,
) Chunker {
	// Check if the chunk index we are looking up already
	// exists in the map of chunks.
	if chunk, ok := chunksByChunkIndex[chunkIndex]; ok {
		return chunk
	}
	// Otherwise, we load the chunk from the database.
	key := s.params.flatSliceID(validatorChunkIndex, chunkIndex)
	data, exists, err := s.serviceCfg.Database.LoadSlasherChunk(context.Background(), kind, key)
	if err != nil {
		panic(err)
	}
	// If the chunk exists in the database, we initialize it from the raw bytes data.
	// If it does not exist, we initialize an empty chunk.
	var existingChunk Chunker
	switch kind {
	case slashertypes.MinSpan:
		if exists {
			existingChunk, err = MinChunkSpansSliceFrom(s.params, data)
		} else {
			existingChunk = EmptyMinSpanChunksSlice(s.params)
		}
	case slashertypes.MaxSpan:
		if exists {
			existingChunk, err = MaxChunkSpansSliceFrom(s.params, data)
		} else {
			existingChunk = EmptyMaxSpanChunksSlice(s.params)
		}
	}
	chunksByChunkIndex[chunkIndex] = existingChunk
	return existingChunk
}

// Group a list of attestations into batches by validator chunk index.
// This way, we can detect on the batch of attestations for each validator chunk index
// concurrently, and also allowing us to effectively use a single 2D chunk
// for slashing detection through this logical grouping.
func (s *Service) groupByValidatorChunkIndex(
	attestations []*slashertypes.CompactAttestation,
) map[uint64][]*slashertypes.CompactAttestation {
	groupedAttestations := make(map[uint64][]*slashertypes.CompactAttestation)
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

// Group attestations by the chunk index their source epoch corresponds to.
func (s *Service) groupByChunkIndex(attestations []*slashertypes.CompactAttestation) map[uint64][]*slashertypes.CompactAttestation {
	attestationsByChunkIndex := make(map[uint64][]*slashertypes.CompactAttestation)
	for _, att := range attestations {
		chunkIdx := s.params.chunkIndex(types.Epoch(att.Source))
		attestationsByChunkIndex[chunkIdx] = append(attestationsByChunkIndex[chunkIdx], att)
	}
	return attestationsByChunkIndex
}
