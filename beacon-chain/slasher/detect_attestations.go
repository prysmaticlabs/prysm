package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

type chunkUpdateOptions struct {
	kind                slashertypes.ChunkKind
	validatorChunkIndex uint64
	currentEpoch        types.Epoch
	chunkIndex          uint64
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection.
// The process is as follows given a list of attestations:
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
func (s *Service) detectSlashableAttestations(
	ctx context.Context,
	attBatch []*slashertypes.CompactAttestation,
	validatorChunkIndex uint64,
	currentEpoch types.Epoch,
) error {
	// Group attestations by chunk index.
	attestationsByChunkIdx := s.groupByChunkIndex(attBatch)
	_ = attestationsByChunkIdx

	_, err := s.updateSpans(ctx, &chunkUpdateOptions{
		kind:                slashertypes.MinSpan,
		validatorChunkIndex: validatorChunkIndex,
		currentEpoch:        currentEpoch,
	})
	if err != nil {
		return err
	}
	_, err = s.updateSpans(ctx, &chunkUpdateOptions{
		kind:                slashertypes.MaxSpan,
		validatorChunkIndex: validatorChunkIndex,
		currentEpoch:        currentEpoch,
	})
	if err != nil {
		return err
	}

	// Update the latest written epoch for all involved validator indices.
	validatorIndices := s.params.validatorIndicesInChunk(validatorChunkIndex)
	return s.serviceCfg.Database.SaveLatestEpochAttestedForValidators(ctx, validatorIndices, currentEpoch)
}

func (s *Service) updateSpans(
	ctx context.Context,
	opts *chunkUpdateOptions,
) (slashertypes.SlashingKind, error) {
	// Update the required chunks with the current epoch.
	validatorIndices := s.params.validatorIndicesInChunk(opts.validatorChunkIndex)
	updatedChunks, err := s.applyCurrentEpochToValidators(ctx, opts, validatorIndices)
	if err != nil {
		return slashertypes.NotSlashable, err
	}

	// Apply the attestations to the related min chunks.
	// TODO: Apply...

	// Write the updated chunks to disk.
	return slashertypes.NotSlashable, s.saveUpdatedChunks(ctx, opts, updatedChunks)
}

func (s *Service) applyCurrentEpochToValidators(
	ctx context.Context,
	opts *chunkUpdateOptions,
	validatorIndices []types.ValidatorIndex,
) (updatedChunks map[uint64]Chunker, err error) {
	epochs, epochsExist, err := s.serviceCfg.Database.LatestEpochAttestedForValidators(ctx, validatorIndices)
	if err != nil {
		return
	}
	updatedChunks = make(map[uint64]Chunker)
	for i := 0; i < len(validatorIndices); i++ {
		validatorIdx := validatorIndices[i]
		lastEpochWritten := epochs[i]
		if !epochsExist[i] {
			lastEpochWritten = types.Epoch(0)
		}
		if err = s.updateChunksWithCurrentEpochForValidator(
			ctx, opts, updatedChunks, lastEpochWritten,
		); err != nil {
			return
		}
	}
	return
}

func (s *Service) updateChunksWithCurrentEpochForValidator(
	ctx context.Context,
	opts *chunkUpdateOptions,
	chunksByChunkIdx map[uint64]Chunker,
	lastEpochWritten types.Epoch,
) error {
	for lastEpochWritten <= opts.currentEpoch {
		chunkIdx := s.params.chunkIndex(lastEpochWritten)
		currentChunk, err := s.loadChunk(chunksByChunkIdx, validatorChunkIdx, chunkIdx, kind)
		if err != nil {
			return err
		}
		for s.params.chunkIndex(epoch) == chunkIdx && epoch <= currentEpoch {
			if err := setChunkDataAtEpoch(
				s.params,
				currentChunk.Chunk(),
				validatorIdx,
				epoch,
				types.Epoch(currentChunk.NeutralElement())+epoch,
			); err != nil {
				return err
			}
			epoch++
		}
		chunksByChunkIdx[chunkIdx] = currentChunk
	}
	return nil
}

func (s *Service) loadChunk(
	ctx context.Context,
	opts *chunkUpdateOptions,
	chunksByChunkIndex map[uint64]Chunker,
) (Chunker, error) {
	// Check if the chunk index we are looking up already
	// exists in the map of chunks.
	if chunk, ok := chunksByChunkIndex[opts.chunkIndex]; ok {
		return chunk, nil
	}
	// Otherwise, we load the chunk from the database.
	key := s.params.flatSliceID(opts.validatorChunkIndex, opts.chunkIndex)
	data, exists, err := s.serviceCfg.Database.LoadSlasherChunk(ctx, opts.kind, key)
	if err != nil {
		return nil, err
	}
	// If the chunk exists in the database, we initialize it from the raw bytes data.
	// If it does not exist, we initialize an empty chunk.
	var existingChunk Chunker
	switch opts.kind {
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
	chunksByChunkIndex[opts.chunkIndex] = existingChunk
	return existingChunk, nil
}

func (s *Service) saveUpdatedChunks(
	ctx context.Context,
	opts *chunkUpdateOptions,
	updatedChunks map[uint64]Chunker,
) error {
	chunkKeys := make([]uint64, 0, len(updatedChunks))
	chunks := make([][]uint16, 0, len(updatedChunks))
	for chunkIdx, chunk := range updatedChunks {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(opts.validatorChunkIndex, chunkIdx))
		chunks = append(chunks, chunk.Chunk())
	}
	return s.serviceCfg.Database.SaveSlasherChunks(ctx, opts.kind, chunkKeys, chunks)
}
