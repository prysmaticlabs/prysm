package slasher

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// A struct encapsulating input arguments to
// functions used for attester slashing detection and
// loading, saving, and updating min/max span chunks.
type chunkUpdateOptions struct {
	kind                slashertypes.ChunkKind
	chunkIndex          uint64
	validatorChunkIndex uint64
	currentEpoch        types.Epoch
	latestEpochWritten  types.Epoch
	validatorIndex      types.ValidatorIndex
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection.
// The process is as follows given a list of attestations:
//
// 1. Group the attestations by chunk index.
// 2. Update the min and max spans for those grouped attestations, check if any slashings are
//    found in the process
// 3. Update the latest written epoch for all validators involved to the current epoch.
//
// This function performs a lot of critical actions and is split into smaller helpers for cleanliness.
func (s *Service) detectSlashableAttestations(
	ctx context.Context,
	opts *chunkUpdateOptions,
	attestations []*slashertypes.CompactAttestation,
) error {
	_, err := s.updateSpans(ctx, &chunkUpdateOptions{
		kind:                slashertypes.MinSpan,
		validatorChunkIndex: opts.validatorChunkIndex,
		currentEpoch:        opts.currentEpoch,
	})
	if err != nil {
		return err
	}
	_, err = s.updateSpans(ctx, &chunkUpdateOptions{
		kind:                slashertypes.MaxSpan,
		validatorChunkIndex: opts.validatorChunkIndex,
		currentEpoch:        opts.currentEpoch,
	})
	if err != nil {
		return err
	}

	// Update the latest written epoch for all involved validator indices.
	validatorIndices := s.params.validatorIndicesInChunk(opts.validatorChunkIndex)
	return s.serviceCfg.Database.SaveLatestEpochAttestedForValidators(ctx, validatorIndices, opts.currentEpoch)
}

// Updates spans and detects any slashable attester offenses along the way.
// 1. Determine the chunks we need to use for updating for the validator indices
//    in a validator chunk index, then retrieve those chunks from the database.
// 2. Using the chunks from step (1):
//      for every attestation by chunk index:
//        for each validator in the attestation's attesting indices:
//          - Check if the attestation is slashable, if so return a slashing object.
// 3. Save the updated chunks to disk.
func (s *Service) updateSpans(
	ctx context.Context,
	opts *chunkUpdateOptions,
) (slashertypes.SlashingKind, error) {
	// Determine the chunk indices we need to use for slashing detection.
	validatorIndices := s.params.validatorIndicesInChunk(opts.validatorChunkIndex)
	chunkIndices, err := s.determineChunksToUpdateForValidators(ctx, opts, validatorIndices)
	if err != nil {
		return slashertypes.NotSlashable, err
	}
	// Load the required chunks from disk.
	chunksByChunkIdx, err := s.loadChunks(ctx, opts, chunkIndices)
	if err != nil {
		return slashertypes.NotSlashable, err
	}

	// Apply the attestations to the related chunks.
	// TODO(#8331): Apply...

	// Write the updated chunks to disk.
	return slashertypes.NotSlashable, s.saveUpdatedChunks(ctx, opts, chunksByChunkIdx)
}

// For a list of validator indices, we retrieve their latest written epoch. Then, for each
// (validator, latest epoch written) pair, we determine the chunks we need to update and
// perform slashing detection with.
func (s *Service) determineChunksToUpdateForValidators(
	ctx context.Context,
	opts *chunkUpdateOptions,
	validatorIndices []types.ValidatorIndex,
) (chunkIndices []uint64, err error) {
	epochs, epochsExist, err := s.serviceCfg.Database.LatestEpochAttestedForValidators(ctx, validatorIndices)
	if err != nil {
		return
	}
	chunkIndicesToUpdate := make(map[uint64]bool)
	for i := 0; i < len(validatorIndices); i++ {
		lastEpochWritten := epochs[i]
		if !epochsExist[i] {
			lastEpochWritten = types.Epoch(0)
		}
		opts.validatorIndex = validatorIndices[i]
		opts.latestEpochWritten = lastEpochWritten
		for opts.latestEpochWritten <= opts.currentEpoch {
			chunkIdx := s.params.chunkIndex(opts.latestEpochWritten)
			chunkIndicesToUpdate[chunkIdx] = true
			opts.latestEpochWritten++
		}
	}
	chunkIndices = make([]uint64, 0, len(chunkIndicesToUpdate))
	for chunkIdx := range chunkIndicesToUpdate {
		chunkIndices = append(chunkIndices, chunkIdx)
	}
	return
}

// Load chunks for a specified list of chunk indices. We attempt to load it from the database.
// If the data exists, then we initialize a chunk of a specified kind. Otherwise, we create
// an empty chunk, add it to our map, and then return it to the caller.
func (s *Service) loadChunks(
	ctx context.Context,
	opts *chunkUpdateOptions,
	chunkIndices []uint64,
) (map[uint64]Chunker, error) {
	chunkKeys := make([]uint64, 0, len(chunkIndices))
	for _, chunkIdx := range chunkIndices {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(opts.validatorChunkIndex, chunkIdx))
	}
	rawChunks, chunksExist, err := s.serviceCfg.Database.LoadSlasherChunks(ctx, opts.kind, chunkKeys)
	if err != nil {
		return nil, err
	}
	chunksByChunkIdx := make(map[uint64]Chunker, len(rawChunks))
	for i := 0; i < len(rawChunks); i++ {
		// If the chunk exists in the database, we initialize it from the raw bytes data.
		// If it does not exist, we initialize an empty chunk.
		var chunk Chunker
		switch opts.kind {
		case slashertypes.MinSpan:
			if chunksExist[i] {
				chunk, err = MinChunkSpansSliceFrom(s.params, rawChunks[i])
			} else {
				chunk = EmptyMinSpanChunksSlice(s.params)
			}
		case slashertypes.MaxSpan:
			if chunksExist[i] {
				chunk, err = MaxChunkSpansSliceFrom(s.params, rawChunks[i])
			} else {
				chunk = EmptyMaxSpanChunksSlice(s.params)
			}
		}
		if err != nil {
			return nil, err
		}
		chunksByChunkIdx[chunkIndices[i]] = chunk
	}
	return chunksByChunkIdx, nil
}

// Saves updated chunks to disk given the required database schema.
func (s *Service) saveUpdatedChunks(
	ctx context.Context,
	opts *chunkUpdateOptions,
	updatedChunksByChunkIdx map[uint64]Chunker,
) error {
	chunkKeys := make([]uint64, 0, len(updatedChunksByChunkIdx))
	chunks := make([][]uint16, 0, len(updatedChunksByChunkIdx))
	for chunkIdx, chunk := range updatedChunksByChunkIdx {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(opts.validatorChunkIndex, chunkIdx))
		chunks = append(chunks, chunk.Chunk())
	}
	return s.serviceCfg.Database.SaveSlasherChunks(ctx, opts.kind, chunkKeys, chunks)
}
