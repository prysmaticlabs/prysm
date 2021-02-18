package slasher

import (
	"context"
	"fmt"

	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// A struct encapsulating input arguments to
// functions used for attester slashing detection and
// loading, saving, and updating min/max span chunks.
type chunkUpdateArgs struct {
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
	args *chunkUpdateArgs,
	attestations []*slashertypes.CompactAttestation,
) error {

	// Group attestations by chunk index.
	groupedAtts := s.groupByChunkIndex(attestations)

	// Update min and max spans and retrieve any detected slashable offenses.
	slashings, err := s.updateSpans(ctx, &chunkUpdateArgs{
		kind:                slashertypes.MinSpan,
		validatorChunkIndex: args.validatorChunkIndex,
		currentEpoch:        args.currentEpoch,
	}, groupedAtts)
	if err != nil {
		return err
	}
	moreSlashings, err := s.updateSpans(ctx, &chunkUpdateArgs{
		kind:                slashertypes.MaxSpan,
		validatorChunkIndex: args.validatorChunkIndex,
		currentEpoch:        args.currentEpoch,
	}, groupedAtts)
	if err != nil {
		return err
	}

	totalSlashings := append(slashings, moreSlashings...)
	if len(totalSlashings) > 0 {
		log.WithField("numSlashings", len(totalSlashings)).Info("Slashable offenses found")
	}
	for _, slashing := range totalSlashings {
		// TODO(#8331): Send over an event feed.
		logSlashingEvent(slashing)
	}

	// Update the latest written epoch for all involved validator indices.
	validatorIndices := s.params.validatorIndicesInChunk(args.validatorChunkIndex)
	return s.serviceCfg.Database.SaveLatestEpochAttestedForValidators(ctx, validatorIndices, args.currentEpoch)
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
	args *chunkUpdateArgs,
	attestationsByChunkIdx map[uint64][]*slashertypes.CompactAttestation,
) ([]*slashertypes.Slashing, error) {
	// Determine the chunk indices we need to use for slashing detection.
	validatorIndices := s.params.validatorIndicesInChunk(args.validatorChunkIndex)
	chunkIndices, err := s.determineChunksToUpdateForValidators(ctx, args, validatorIndices)
	if err != nil {
		return nil, err
	}
	// Load the required chunks from disk.
	chunksByChunkIdx, err := s.loadChunks(ctx, args, chunkIndices)
	if err != nil {
		return nil, err
	}

	// Apply the attestations to the related chunks and find any
	// slashings along the way.
	slashings := make([]*slashertypes.Slashing, 0)
	for _, attestationBatch := range attestationsByChunkIdx {
		for _, att := range attestationBatch {
			for _, validatorIdx := range att.AttestingIndices {
				args.validatorIndex = types.ValidatorIndex(validatorIdx)
				computedValidatorChunkIdx := s.params.validatorChunkIndex(args.validatorIndex)
				if args.validatorChunkIndex != computedValidatorChunkIdx {
					continue
				}
				slashing, err := s.applyAttestationForValidator(
					ctx,
					args,
					chunksByChunkIdx,
					att,
				)
				if err != nil {
					return nil, err
				}
				if slashing != nil {
					slashings = append(slashings, slashing)
				}
			}
		}
	}

	// Write the updated chunks to disk.
	return slashings, s.saveUpdatedChunks(ctx, args, chunksByChunkIdx)
}

// For a list of validator indices, we retrieve their latest written epoch. Then, for each
// (validator, latest epoch written) pair, we determine the chunks we need to update and
// perform slashing detection with.
func (s *Service) determineChunksToUpdateForValidators(
	ctx context.Context,
	args *chunkUpdateArgs,
	validatorIndices []types.ValidatorIndex,
) (chunkIndices []uint64, err error) {
	attestedEpochs, err := s.serviceCfg.Database.LatestEpochAttestedForValidators(ctx, validatorIndices)
	if err != nil {
		return
	}
	chunkIndicesToUpdate := make(map[uint64]bool)

	// Initialize the latest epoch attested for each validator to 0.
	latestEpochAttestedByValidator := make(map[types.ValidatorIndex]types.Epoch)
	for _, valIdx := range validatorIndices {
		latestEpochAttestedByValidator[valIdx] = 0
	}
	// If we have indeed attested an epoch, we update that value in
	// the map from the previous step.
	for _, attestedEpoch := range attestedEpochs {
		latestEpochAttestedByValidator[attestedEpoch.ValidatorIndex] = attestedEpoch.Epoch
	}

	// For every single validator and their latest epoch attested, we determine
	// the chunk indices we need to update based on all the chunks between the latest
	// epoch written and the current epoch, inclusive.
	for valIdx, epoch := range latestEpochAttestedByValidator {
		args.validatorIndex = valIdx
		args.latestEpochWritten = epoch
		for args.latestEpochWritten <= args.currentEpoch {
			chunkIdx := s.params.chunkIndex(args.latestEpochWritten)
			chunkIndicesToUpdate[chunkIdx] = true
			args.latestEpochWritten++
		}
	}
	chunkIndices = make([]uint64, 0, len(chunkIndicesToUpdate))
	for chunkIdx := range chunkIndicesToUpdate {
		chunkIndices = append(chunkIndices, chunkIdx)
	}
	return
}

// Checks if an incoming attestation is slashable based on the validator chunk it
// corresponds to. If a slashable offense is found, we return it to the caller.
// If not, then update every single chunk the attestation covers, starting from its
// source epoch up to its target.
func (s *Service) applyAttestationForValidator(
	ctx context.Context,
	args *chunkUpdateArgs,
	chunksByChunkIdx map[uint64]Chunker,
	attestation *slashertypes.CompactAttestation,
) (*slashertypes.Slashing, error) {
	sourceEpoch := attestation.Source
	targetEpoch := attestation.Target
	chunkIdx := s.params.chunkIndex(sourceEpoch)
	chunk, ok := chunksByChunkIdx[chunkIdx]
	if !ok {
		// It is possible we receive an attestation corresponding to a chunk index
		// we are not yet updating, so we ignore.
		return nil, nil
	}

	// Check slashable, if so, return the slashing.
	slashing, err := chunk.CheckSlashable(
		ctx,
		s.serviceCfg.Database,
		args.validatorIndex,
		attestation,
	)
	if err != nil {
		return nil, err
	}
	if slashing != nil && slashing.Kind != slashertypes.NotSlashable {
		return slashing, nil
	}

	// Get the first start epoch for the chunk. If it does not exist or
	// is not possible based on the input arguments, do not continue with the update.
	startEpoch, exists := chunk.StartEpoch(sourceEpoch, args.currentEpoch)
	if !exists {
		return nil, nil
	}

	// Given a single attestation could span across multiple chunks
	// for a validator min or max span, we attempt to update the current chunk
	// for the source epoch of the attestation. If the update function tells
	// us we need to proceed to the next chunk, we continue by determining
	// the start epoch of the next chunk. We exit once no longer need to
	// keep updating chunks.
	for {
		chunkIdx = s.params.chunkIndex(startEpoch)
		chunk, ok = chunksByChunkIdx[chunkIdx]
		if !ok {
			return nil, fmt.Errorf("chunk at chunk index %d not found", chunkIdx)
		}
		keepGoing, err := chunk.Update(
			&chunkUpdateArgs{
				chunkIndex:     chunkIdx,
				currentEpoch:   args.currentEpoch,
				validatorIndex: args.validatorIndex,
			},
			startEpoch,
			targetEpoch,
		)
		if err != nil {
			return nil, err
		}
		// We update the chunksByChunkIdx map with the chunk we just updated.
		chunksByChunkIdx[chunkIdx] = chunk
		if !keepGoing {
			break
		}
		// Move to first epoch of next chunk if needed.
		startEpoch = chunk.NextChunkStartEpoch(startEpoch)
	}
	return nil, nil
}

// Load chunks for a specified list of chunk indices. We attempt to load it from the database.
// If the data exists, then we initialize a chunk of a specified kind. Otherwise, we create
// an empty chunk, add it to our map, and then return it to the caller.
func (s *Service) loadChunks(
	ctx context.Context,
	args *chunkUpdateArgs,
	chunkIndices []uint64,
) (map[uint64]Chunker, error) {
	chunkKeys := make([]uint64, 0, len(chunkIndices))
	for _, chunkIdx := range chunkIndices {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(args.validatorChunkIndex, chunkIdx))
	}
	rawChunks, chunksExist, err := s.serviceCfg.Database.LoadSlasherChunks(ctx, args.kind, chunkKeys)
	if err != nil {
		return nil, err
	}
	chunksByChunkIdx := make(map[uint64]Chunker, len(rawChunks))
	for i := 0; i < len(rawChunks); i++ {
		// If the chunk exists in the database, we initialize it from the raw bytes data.
		// If it does not exist, we initialize an empty chunk.
		var chunk Chunker
		switch args.kind {
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
	args *chunkUpdateArgs,
	updatedChunksByChunkIdx map[uint64]Chunker,
) error {
	chunkKeys := make([]uint64, 0, len(updatedChunksByChunkIdx))
	chunks := make([][]uint16, 0, len(updatedChunksByChunkIdx))
	for chunkIdx, chunk := range updatedChunksByChunkIdx {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(args.validatorChunkIndex, chunkIdx))
		chunks = append(chunks, chunk.Chunk())
	}
	return s.serviceCfg.Database.SaveSlasherChunks(ctx, args.kind, chunkKeys, chunks)
}
