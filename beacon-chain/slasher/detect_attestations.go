package slasher

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	slashertypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Takes in a list of indexed attestation wrappers and returns any
// found attester slashings to the caller.
func (s *Service) checkSlashableAttestations(
	ctx context.Context, currentEpoch primitives.Epoch, atts []*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	totalStart := time.Now()

	slashings := make([]*ethpb.AttesterSlashing, 0)

	// Double votes
	log.Debug("Checking for double votes")
	start := time.Now()
	doubleVoteSlashings, err := s.checkDoubleVotes(ctx, atts)
	if err != nil {
		return nil, errors.Wrap(err, "could not check slashable double votes")
	}

	log.WithField("elapsed", time.Since(start)).Debug("Done checking double votes")

	slashings = append(slashings, doubleVoteSlashings...)

	// Surrounding / surrounded votes
	groupedByValidatorChunkIndexAtts := s.groupByValidatorChunkIndex(atts)
	log.WithField("numBatches", len(groupedByValidatorChunkIndexAtts)).Debug("Batching attestations by validator chunk index")
	groupsCount := len(groupedByValidatorChunkIndexAtts)

	surroundStart := time.Now()

	for validatorChunkIndex, attestations := range groupedByValidatorChunkIndexAtts {
		// The fact that we use always slashertypes.MinSpan is probably the root cause of
		// https://github.com/prysmaticlabs/prysm/issues/13591
		attSlashings, err := s.checkSurrounds(ctx, attestations, slashertypes.MinSpan, currentEpoch, validatorChunkIndex)
		if err != nil {
			return nil, err
		}

		slashings = append(slashings, attSlashings...)

		indices := s.params.validatorIndexesInChunk(validatorChunkIndex)
		for _, idx := range indices {
			s.latestEpochWrittenForValidator[idx] = currentEpoch
		}
	}

	surroundElapsed := time.Since(surroundStart)
	totalElapsed := time.Since(totalStart)

	fields := logrus.Fields{
		"numAttestations":                 len(atts),
		"numBatchesByValidatorChunkIndex": groupsCount,
		"elapsed":                         totalElapsed,
	}

	if groupsCount > 0 {
		avgProcessingTimePerBatch := surroundElapsed / time.Duration(groupsCount)
		fields["avgBatchProcessingTime"] = avgProcessingTimePerBatch
	}

	log.WithFields(fields).Info("Done checking slashable attestations")

	if len(slashings) > 0 {
		log.WithField("numSlashings", len(slashings)).Warn("Slashable attestation offenses found")
	}

	return slashings, nil
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection.
// The process is as follows given a list of attestations:
//
//  1. Group the attestations by chunk index.
//  2. Update the min and max spans for those grouped attestations, check if any slashings are
//     found in the process
//  3. Update the latest written epoch for all validators involved to the current epoch.
//
// This function performs a lot of critical actions and is split into smaller helpers for cleanliness.
func (s *Service) checkSurrounds(
	ctx context.Context,
	attestations []*slashertypes.IndexedAttestationWrapper,
	chunkKind slashertypes.ChunkKind,
	currentEpoch primitives.Epoch,
	validatorChunkIndex uint64,
) ([]*ethpb.AttesterSlashing, error) {
	// Map of updated chunks by chunk index, which will be saved at the end.
	updatedChunks := make(map[uint64]Chunker)
	groupedByChunkIndexAtts := s.groupByChunkIndex(attestations)
	validatorIndexes := s.params.validatorIndexesInChunk(validatorChunkIndex)

	// Update the min/max span chunks for the change of current epoch.
	for _, validatorIndex := range validatorIndexes {
		// This function modifies `updatedChunks` in place.
		if err := s.epochUpdateForValidator(ctx, updatedChunks, validatorChunkIndex, chunkKind, currentEpoch, validatorIndex); err != nil {
			return nil, errors.Wrapf(err, "could not update validator index chunks %d", validatorIndex)
		}
	}

	// Update min and max spans and retrieve any detected slashable offenses.
	surroundingSlashings, err := s.updateSpans(ctx, updatedChunks, groupedByChunkIndexAtts, slashertypes.MinSpan, validatorChunkIndex, currentEpoch)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update min attestation spans for validator chunk index %d", validatorChunkIndex)
	}

	surroundedSlashings, err := s.updateSpans(ctx, updatedChunks, groupedByChunkIndexAtts, slashertypes.MaxSpan, validatorChunkIndex, currentEpoch)
	if err != nil {
		return nil, errors.Wrapf(err, "could not update max attestation spans for validator chunk index %d", validatorChunkIndex)
	}

	slashings := make([]*ethpb.AttesterSlashing, 0, len(surroundingSlashings)+len(surroundedSlashings))
	slashings = append(slashings, surroundingSlashings...)
	slashings = append(slashings, surroundedSlashings...)
	if err := s.saveUpdatedChunks(ctx, updatedChunks, chunkKind, validatorChunkIndex); err != nil {
		return nil, err
	}
	return slashings, nil
}

// Check for double votes in our database given a list of incoming attestations.
func (s *Service) checkDoubleVotes(
	ctx context.Context, attestations []*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleVotesOnDisk")
	defer span.End()
	doubleVotes, err := s.serviceCfg.Database.CheckAttesterDoubleVotes(
		ctx, attestations,
	)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve potential double votes from disk")
	}
	doubleVoteSlashings := make([]*ethpb.AttesterSlashing, 0)
	for _, doubleVote := range doubleVotes {
		doubleVotesTotal.Inc()
		doubleVoteSlashings = append(doubleVoteSlashings, &ethpb.AttesterSlashing{
			Attestation_1: doubleVote.PrevAttestationWrapper.IndexedAttestation,
			Attestation_2: doubleVote.AttestationWrapper.IndexedAttestation,
		})
	}
	return doubleVoteSlashings, nil
}

// This function updates `updatedChunks`, representing the slashing spans for a given validator for
// a change in epoch since the last epoch we have recorded for the validator.
// For example, if the last epoch a validator has written is N, and the current epoch is N+5,
// we update entries in the slashing spans with their neutral element for epochs N+1 to N+4.
// This also puts any loaded chunks in a map used as a cache for further processing and minimizing
// database reads later on.
func (s *Service) epochUpdateForValidator(
	ctx context.Context,
	updatedChunks map[uint64]Chunker,
	validatorChunkIndex uint64,
	chunkKind slashertypes.ChunkKind,
	currentEpoch primitives.Epoch,
	validatorIndex primitives.ValidatorIndex,
) error {
	var err error

	latestEpochWritten, ok := s.latestEpochWrittenForValidator[validatorIndex]
	if !ok {
		return nil
	}

	for latestEpochWritten <= currentEpoch {
		chunkIndex := s.params.chunkIndex(latestEpochWritten)

		currentChunk, ok := updatedChunks[chunkIndex]
		if !ok {
			currentChunk, err = s.getChunk(ctx, chunkKind, validatorChunkIndex, chunkIndex)
			if err != nil {
				return errors.Wrap(err, "could not get chunk")
			}
		}

		for s.params.chunkIndex(latestEpochWritten) == chunkIndex && latestEpochWritten <= currentEpoch {
			if err := setChunkRawDistance(
				s.params,
				currentChunk.Chunk(),
				validatorIndex,
				latestEpochWritten,
				currentChunk.NeutralElement(),
			); err != nil {
				return err
			}

			updatedChunks[chunkIndex] = currentChunk
			latestEpochWritten++
		}
	}

	return nil
}

// Updates spans and detects any slashable attester offenses along the way.
//  1. Determine the chunks we need to use for updating for the validator indices
//     in a validator chunk index, then retrieve those chunks from the database.
//  2. Using the chunks from step (1):
//     for every attestation by chunk index:
//     for each validator in the attestation's attesting indices:
//     - Check if the attestation is slashable, if so return a slashing object.
//  3. Save the updated chunks to disk.
func (s *Service) updateSpans(
	ctx context.Context,
	updatedChunks map[uint64]Chunker,
	attestationsByChunkIdx map[uint64][]*slashertypes.IndexedAttestationWrapper,
	kind slashertypes.ChunkKind,
	validatorChunkIndex uint64,
	currentEpoch primitives.Epoch,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.updateSpans")
	defer span.End()

	// Apply the attestations to the related chunks and find any
	// slashings along the way.
	slashings := make([]*ethpb.AttesterSlashing, 0)
	for _, attestations := range attestationsByChunkIdx {
		for _, attestation := range attestations {
			for _, validatorIdx := range attestation.IndexedAttestation.AttestingIndices {
				validatorIndex := primitives.ValidatorIndex(validatorIdx)
				computedValidatorChunkIdx := s.params.validatorChunkIndex(validatorIndex)

				// Every validator chunk index represents a range of validators.
				// It is possible that the validator index in this loop iteration is
				// not part of the validator chunk index we are updating chunks for.
				//
				// For example, if there are 4 validators per validator chunk index,
				// then validator chunk index 0 contains validator indices [0, 1, 2, 3]
				// If we see an attestation with attesting indices [3, 4, 5] and we are updating
				// chunks for validator chunk index 0, only validator index 3 should make
				// it past this line.
				if validatorChunkIndex != computedValidatorChunkIdx {
					continue
				}

				slashing, err := s.applyAttestationForValidator(
					ctx, updatedChunks, attestation, kind, validatorChunkIndex, validatorIndex, currentEpoch,
				)

				if err != nil {
					return nil, errors.Wrapf(err, "could not apply attestation for validator index %d", validatorIndex)
				}

				if slashing == nil {
					continue
				}

				slashings = append(slashings, slashing)
			}
		}
	}

	// Write the updated chunks to disk.
	return slashings, nil
}

// Checks if an incoming attestation is slashable based on the validator chunk it
// corresponds to. If a slashable offense is found, we return it to the caller.
// If not, then update every single chunk the attestation covers, starting from its
// source epoch up to its target.
func (s *Service) applyAttestationForValidator(
	ctx context.Context,
	chunksByChunkIdx map[uint64]Chunker,
	attestation *slashertypes.IndexedAttestationWrapper,
	chunkKind slashertypes.ChunkKind,
	validatorChunkIndex uint64,
	validatorIndex primitives.ValidatorIndex,
	currentEpoch primitives.Epoch,
) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.applyAttestationForValidator")
	defer span.End()

	var err error

	sourceEpoch := attestation.IndexedAttestation.Data.Source.Epoch
	targetEpoch := attestation.IndexedAttestation.Data.Target.Epoch

	attestationDistance.Observe(float64(targetEpoch) - float64(sourceEpoch))
	chunkIndex := s.params.chunkIndex(sourceEpoch)

	chunk, ok := chunksByChunkIdx[chunkIndex]
	if !ok {
		chunk, err = s.getChunk(ctx, chunkKind, validatorChunkIndex, chunkIndex)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get chunk at index %d", chunkIndex)
		}
	}

	// Check slashable, if so, return the slashing.
	slashing, err := chunk.CheckSlashable(
		ctx,
		s.serviceCfg.Database,
		validatorIndex,
		attestation,
	)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not check if attestation for validator index %d is slashable",
			validatorIndex,
		)
	}
	if slashing != nil {
		return slashing, nil
	}

	// Get the first start epoch for the chunk. If it does not exist or
	// is not possible based on the input arguments, do not continue with the update.
	startEpoch, exists := chunk.StartEpoch(sourceEpoch, currentEpoch)
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
		chunkIndex = s.params.chunkIndex(startEpoch)

		chunk, ok := chunksByChunkIdx[chunkIndex]
		if !ok {
			chunk, err = s.getChunk(ctx, chunkKind, validatorChunkIndex, chunkIndex)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get chunk at index %d", chunkIndex)
			}
		}

		keepGoing, err := chunk.Update(
			&chunkUpdateArgs{
				chunkIndex:   chunkIndex,
				currentEpoch: currentEpoch,
			},
			validatorIndex,
			startEpoch,
			targetEpoch,
		)

		if err != nil {
			return nil, errors.Wrapf(
				err,
				"could not update chunk at chunk index %d for validator index %d and current epoch %d",
				chunkIndex,
				validatorIndex,
				currentEpoch,
			)
		}

		// We update the chunksByChunkIdx map with the chunk we just updated.
		chunksByChunkIdx[chunkIndex] = chunk
		if !keepGoing {
			break
		}

		// Move to first epoch of next chunk if needed.
		startEpoch = chunk.NextChunkStartEpoch(startEpoch)
	}

	return nil, nil
}

// Retrieve a chunk from database from database.
func (s *Service) getChunk(
	ctx context.Context,
	chunkKind slashertypes.ChunkKind,
	validatorChunkIndex uint64,
	chunkIndex uint64,
) (Chunker, error) {
	// We can ensure we load the appropriate chunk we need by fetching from the DB.
	diskChunks, err := s.loadChunks(ctx, validatorChunkIndex, chunkKind, []uint64{chunkIndex})
	if err != nil {
		return nil, errors.Wrapf(err, "could not load chunk at index %d", chunkIndex)
	}

	if chunk, ok := diskChunks[chunkIndex]; ok {
		return chunk, nil
	}

	return nil, fmt.Errorf("could not retrieve chunk at chunk index %d from disk", chunkIndex)
}

// Load chunks for a specified list of chunk indices. We attempt to load it from the database.
// If the data exists, then we initialize a chunk of a specified kind. Otherwise, we create
// an empty chunk, add it to our map, and then return it to the caller.
func (s *Service) loadChunks(
	ctx context.Context,
	validatorChunkIndex uint64,
	chunkKind slashertypes.ChunkKind,
	chunkIndices []uint64,
) (map[uint64]Chunker, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.loadChunks")
	defer span.End()

	chunkKeys := make([][]byte, 0, len(chunkIndices))
	for _, chunkIdx := range chunkIndices {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(validatorChunkIndex, chunkIdx))
	}

	rawChunks, chunksExist, err := s.serviceCfg.Database.LoadSlasherChunks(ctx, chunkKind, chunkKeys)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load slasher chunk index")
	}

	chunksByChunkIdx := make(map[uint64]Chunker, len(rawChunks))
	for i := 0; i < len(rawChunks); i++ {
		// If the chunk exists in the database, we initialize it from the raw bytes data.
		// If it does not exist, we initialize an empty chunk.
		var (
			chunk Chunker
			err   error
		)

		chunkExists := chunksExist[i]

		switch chunkKind {
		case slashertypes.MinSpan:
			if chunkExists {
				chunk, err = MinChunkSpansSliceFrom(s.params, rawChunks[i])
				break
			}
			chunk = EmptyMinSpanChunksSlice(s.params)

		case slashertypes.MaxSpan:
			if chunkExists {
				chunk, err = MaxChunkSpansSliceFrom(s.params, rawChunks[i])
				break
			}
			chunk = EmptyMaxSpanChunksSlice(s.params)
		}

		if err != nil {
			return nil, errors.Wrap(err, "could not initialize chunk")
		}

		chunksByChunkIdx[chunkIndices[i]] = chunk
	}

	return chunksByChunkIdx, nil
}

// Saves updated chunks to disk given the required database schema.
func (s *Service) saveUpdatedChunks(
	ctx context.Context,
	updatedChunksByChunkIdx map[uint64]Chunker,
	chunkKind slashertypes.ChunkKind,
	validatorChunkIndex uint64,
) error {
	ctx, span := trace.StartSpan(ctx, "Slasher.saveUpdatedChunks")
	defer span.End()
	chunkKeys := make([][]byte, 0, len(updatedChunksByChunkIdx))
	chunks := make([][]uint16, 0, len(updatedChunksByChunkIdx))
	for chunkIdx, chunk := range updatedChunksByChunkIdx {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(validatorChunkIndex, chunkIdx))
		chunks = append(chunks, chunk.Chunk())
	}
	chunksSavedTotal.Add(float64(len(chunks)))
	return s.serviceCfg.Database.SaveSlasherChunks(ctx, chunkKind, chunkKeys, chunks)
}
