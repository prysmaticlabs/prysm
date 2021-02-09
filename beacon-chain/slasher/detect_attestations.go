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
	validatorChunkIndex uint64
	chunkIndex          uint64
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
// 4. Update the latest written epoch for all validators involved to the current epoch.
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
// 1. Update the latest written epoch for all validator indices involved up and
//    including the current epoch, return the updated chunks by chunk index.
// 2. Using the chunks from step (1):
//      for every attestation by chunk index:
//        for each validator in its attesting indices:
//          - Check if the attestation is slashable, if so return a slashing object.
// 3. Save the updated chunks to disk.
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

	// Apply the attestations to the related chunks.
	// TODO(#8331): Apply...

	// Write the updated chunks to disk.
	return slashertypes.NotSlashable, s.saveUpdatedChunks(ctx, opts, updatedChunks)
}

// For a list of validator indices, we retrieve their latest written epoch. Then, for each
// (validator, latest epoch written) pair, we update chunks with neutral values from the
// latest written epoch up to and including the current epoch.
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
		lastEpochWritten := epochs[i]
		if !epochsExist[i] {
			lastEpochWritten = types.Epoch(0)
		}
		opts.validatorIndex = validatorIndices[i]
		opts.latestEpochWritten = lastEpochWritten
		if err = s.updateChunksWithCurrentEpochForValidator(
			ctx, opts, updatedChunks,
		); err != nil {
			return
		}
	}
	return
}

// Updates every chunk for a validator index from that validator's
// latest written epoch up to and including the current epoch to
// neutral element values. For min chunks, this value is MaxUint16
// and for max chunks the neutral element is 0.
func (s *Service) updateChunksWithCurrentEpochForValidator(
	ctx context.Context,
	opts *chunkUpdateOptions,
	chunksByChunkIdx map[uint64]Chunker,
) error {
	for opts.latestEpochWritten <= opts.currentEpoch {
		chunkIdx := s.params.chunkIndex(opts.latestEpochWritten)
		currentChunk, err := s.loadChunk(ctx, opts, chunksByChunkIdx)
		if err != nil {
			return err
		}
		for s.params.chunkIndex(opts.latestEpochWritten) == chunkIdx && opts.latestEpochWritten <= opts.currentEpoch {
			if err := setChunkRawDistance(
				s.params,
				currentChunk.Chunk(),
				opts.validatorIndex,
				opts.latestEpochWritten,
				currentChunk.NeutralElement(),
			); err != nil {
				return err
			}
			opts.latestEpochWritten++
		}
		chunksByChunkIdx[chunkIdx] = currentChunk
	}
	return nil
}

// Load chunk, when given a map of chunks by chunk index, checks if the chunk we want to retrieve
// is already in this map and returns it. If not, we attempt to load it from the database.
// If the data exists, then we initialize a chunk of a specified kind. Otherwise, we create
// an empty chunk, add it to our map, and then return it to the caller.
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

// Saves updated chunks to disk given the required database schema.
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
