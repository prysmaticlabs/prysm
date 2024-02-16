package slasher

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	slashertypes "github.com/prysmaticlabs/prysm/v5/beacon-chain/slasher/types"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// Takes in a list of indexed attestation wrappers and returns any
// found attester slashings to the caller.
func (s *Service) checkSlashableAttestations(
	ctx context.Context, currentEpoch primitives.Epoch, atts []*slashertypes.IndexedAttestationWrapper,
) (map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing, error) {
	start := time.Now()

	slashings := map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing{}

	// Double votes
	doubleVoteSlashings, err := s.checkDoubleVotes(ctx, atts)
	if err != nil {
		return nil, errors.Wrap(err, "could not check slashable double votes")
	}

	log.WithField("elapsed", time.Since(start)).Debug("Done checking double votes")

	for root, slashing := range doubleVoteSlashings {
		slashings[root] = slashing
	}

	// Save the attestation records to our database.
	// If multiple attestations are provided for the same validator index + target epoch combination,
	// then the first (validator index + target epoch) => signing root) link is kept into the database.
	if err := s.serviceCfg.Database.SaveAttestationRecordsForValidators(ctx, atts); err != nil {
		return nil, errors.Wrap(err, couldNotSaveAttRecord)
	}

	// Surrounding / surrounded votes
	surroundSlashings, err := s.checkSurroundVotes(ctx, atts, currentEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "could not check slashable surround votes")
	}

	for root, slashing := range surroundSlashings {
		slashings[root] = slashing
	}

	elapsed := time.Since(start)

	fields := logrus.Fields{
		"numAttestations": len(atts),
		"elapsed":         elapsed,
	}

	log.WithFields(fields).Info("Done checking slashable attestations")

	if len(slashings) > 0 {
		log.WithField("numSlashings", len(slashings)).Warn("Slashable attestation offenses found")
	}

	return slashings, nil
}

// Check for surrounding and surrounded votes in our database given a list of incoming attestations.
func (s *Service) checkSurroundVotes(
	ctx context.Context,
	attWrappers []*slashertypes.IndexedAttestationWrapper,
	currentEpoch primitives.Epoch,
) (map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing, error) {
	slashings := map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing{}

	// Group attestation wrappers by validator chunk index.
	attWrappersByValidatorChunkIndex := s.groupByValidatorChunkIndex(attWrappers)

	for validatorChunkIndex, attWrappers := range attWrappersByValidatorChunkIndex {
		minChunkByChunkIndex, err := s.updatedChunkByChunkIndex(ctx, slashertypes.MinSpan, currentEpoch, validatorChunkIndex)
		if err != nil {
			return nil, errors.Wrap(err, "could not update updatedMinChunks")
		}

		maxChunkByChunkIndex, err := s.updatedChunkByChunkIndex(ctx, slashertypes.MaxSpan, currentEpoch, validatorChunkIndex)
		if err != nil {
			return nil, errors.Wrap(err, "could not update updatedMaxChunks")
		}

		// Group (already grouped by validator chunk index) attestation wrappers by chunk index.
		attWrappersByChunkIndex := s.groupByChunkIndex(attWrappers)

		// Check for surrounding votes.
		surroundingSlashings, err := s.updateSpans(ctx, minChunkByChunkIndex, attWrappersByChunkIndex, slashertypes.MinSpan, validatorChunkIndex, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not update min attestation spans for validator chunk index %d", validatorChunkIndex)
		}

		for root, slashing := range surroundingSlashings {
			slashings[root] = slashing
		}

		// Check for surrounded votes.
		surroundedSlashings, err := s.updateSpans(ctx, maxChunkByChunkIndex, attWrappersByChunkIndex, slashertypes.MaxSpan, validatorChunkIndex, currentEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "could not update max attestation spans for validator chunk index %d", validatorChunkIndex)
		}

		for root, slashing := range surroundedSlashings {
			slashings[root] = slashing
		}

		// Save updated chunks into the database.
		if err := s.saveUpdatedChunks(ctx, minChunkByChunkIndex, slashertypes.MinSpan, validatorChunkIndex); err != nil {
			return nil, errors.Wrap(err, "could not save chunks for min spans")
		}

		if err := s.saveUpdatedChunks(ctx, maxChunkByChunkIndex, slashertypes.MaxSpan, validatorChunkIndex); err != nil {
			return nil, errors.Wrap(err, "could not save chunks for max spans")
		}

		// Update the latest written epoch for all validators involved to the current chunk.
		indices := s.params.validatorIndexesInChunk(validatorChunkIndex)
		for _, idx := range indices {
			s.latestEpochWrittenForValidator[idx] = currentEpoch
		}
	}

	return slashings, nil
}

// Check for double votes in our database given a list of incoming attestations.
func (s *Service) checkDoubleVotes(
	ctx context.Context, incomingAttWrappers []*slashertypes.IndexedAttestationWrapper,
) (map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleVotesOnDisk")
	defer span.End()

	type attestationInfo struct {
		validatorIndex uint64
		epoch          primitives.Epoch
	}

	slashings := map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing{}

	// Check each incoming attestation for double votes against other incoming attestations.
	existingAttWrappers := make(map[attestationInfo]*slashertypes.IndexedAttestationWrapper)

	for _, incomingAttWrapper := range incomingAttWrappers {
		targetEpoch := incomingAttWrapper.IndexedAttestation.Data.Target.Epoch

		for _, validatorIndex := range incomingAttWrapper.IndexedAttestation.AttestingIndices {
			info := attestationInfo{
				validatorIndex: validatorIndex,
				epoch:          targetEpoch,
			}

			existingAttWrapper, ok := existingAttWrappers[info]
			if !ok {
				// This is the first attestation for this `validator index x epoch` combination.
				// There is no double vote. This attestation is memoized for future checks.
				existingAttWrappers[info] = incomingAttWrapper
				continue
			}

			if existingAttWrapper.DataRoot == incomingAttWrapper.DataRoot {
				// Both attestations are the same, this is not a double vote.
				continue
			}

			// There is two different attestations for the same `validator index x epoch` combination.
			// This is a double vote.
			doubleVotesTotal.Inc()

			slashing := &ethpb.AttesterSlashing{
				Attestation_1: existingAttWrapper.IndexedAttestation,
				Attestation_2: incomingAttWrapper.IndexedAttestation,
			}

			// Ensure the attestation with the lower data root is the first attestation.
			// It will be useful for comparing with other double votes.
			if bytes.Compare(existingAttWrapper.DataRoot[:], incomingAttWrapper.DataRoot[:]) > 0 {
				slashing = &ethpb.AttesterSlashing{
					Attestation_1: incomingAttWrapper.IndexedAttestation,
					Attestation_2: existingAttWrapper.IndexedAttestation,
				}
			}

			root, err := slashing.HashTreeRoot()
			if err != nil {
				return nil, errors.Wrap(err, "could not hash tree root for attester slashing")
			}

			slashings[root] = slashing
		}
	}

	// Check each incoming attestation for double votes against the database.
	doubleVotes, err := s.serviceCfg.Database.CheckAttesterDoubleVotes(ctx, incomingAttWrappers)

	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve potential double votes from disk")
	}

	for _, doubleVote := range doubleVotes {
		doubleVotesTotal.Inc()

		wrapper_1 := doubleVote.Wrapper_1
		wrapper_2 := doubleVote.Wrapper_2

		slashing := &ethpb.AttesterSlashing{
			Attestation_1: wrapper_1.IndexedAttestation,
			Attestation_2: wrapper_2.IndexedAttestation,
		}

		// Ensure the attestation with the lower data root is the first attestation.
		if bytes.Compare(wrapper_1.DataRoot[:], wrapper_2.DataRoot[:]) > 0 {
			slashing = &ethpb.AttesterSlashing{
				Attestation_1: wrapper_2.IndexedAttestation,
				Attestation_2: wrapper_1.IndexedAttestation,
			}
		}

		root, err := slashing.HashTreeRoot()
		if err != nil {
			return nil, errors.Wrap(err, "could not hash tree root for attester slashing")
		}

		slashings[root] = slashing
	}

	return slashings, nil
}

// updatedChunkByChunkIndex loads the chunks from the database for validators corresponding to
// the `validatorChunkIndex`.
// It then updates the chunks with the neutral element for corresponding validators from
// the epoch just after the latest epoch written to the current epoch.
// A mapping between chunk index and chunk is returned to the caller.
func (s *Service) updatedChunkByChunkIndex(
	ctx context.Context,
	chunkKind slashertypes.ChunkKind,
	currentEpoch primitives.Epoch,
	validatorChunkIndex uint64,
) (map[uint64]Chunker, error) {
	chunkByChunkIndex := map[uint64]Chunker{}

	validatorIndexes := s.params.validatorIndexesInChunk(validatorChunkIndex)
	for _, validatorIndex := range validatorIndexes {
		// Retrieve the latest epoch written for the validator.
		latestEpochWritten, ok := s.latestEpochWrittenForValidator[validatorIndex]

		// Start from the epoch just after the latest epoch written.
		epochToWrite, err := latestEpochWritten.SafeAdd(1)
		if err != nil {
			return nil, errors.Wrap(err, "could not add 1 to latest epoch written")
		}

		if !ok {
			epochToWrite = 0
		}

		// It is useless to update more than `historyLength` epochs, since
		// the chunks are circular and we will be overwritten at least one.
		if currentEpoch-epochToWrite >= s.params.historyLength {
			epochToWrite = currentEpoch + 1 - s.params.historyLength
		}

		for epochToWrite <= currentEpoch {
			// Get the chunk index for the latest epoch written.
			chunkIndex := s.params.chunkIndex(epochToWrite)

			// Get the chunk corresponding to the chunk index from the `chunkByChunkIndex` map.
			currentChunk, ok := chunkByChunkIndex[chunkIndex]
			if !ok {
				// If the chunk is not in the map, retrieve it from the database.
				currentChunk, err = s.getChunkFromDatabase(ctx, chunkKind, validatorChunkIndex, chunkIndex)
				if err != nil {
					return nil, errors.Wrap(err, "could not get chunk")
				}
			}

			// Update the current chunk with the neutral element for the validator index for the latest epoch written.
			for s.params.chunkIndex(epochToWrite) == chunkIndex && epochToWrite <= currentEpoch {
				if err := setChunkRawDistance(
					s.params,
					currentChunk.Chunk(),
					validatorIndex,
					epochToWrite,
					currentChunk.NeutralElement(),
				); err != nil {
					return nil, err
				}

				epochToWrite++
			}

			chunkByChunkIndex[chunkIndex] = currentChunk
		}
	}

	return chunkByChunkIndex, nil
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
	attWrapperByChunkIdx map[uint64][]*slashertypes.IndexedAttestationWrapper,
	kind slashertypes.ChunkKind,
	validatorChunkIndex uint64,
	currentEpoch primitives.Epoch,
) (map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.updateSpans")
	defer span.End()

	// Apply the attestations to the related chunks and find any
	// slashings along the way.
	slashings := map[[fieldparams.RootLength]byte]*ethpb.AttesterSlashing{}

	for _, attWrappers := range attWrapperByChunkIdx {
		for _, attWrapper := range attWrappers {
			for _, validatorIdx := range attWrapper.IndexedAttestation.AttestingIndices {
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
					ctx, updatedChunks, attWrapper, kind, validatorChunkIndex, validatorIndex, currentEpoch,
				)

				if err != nil {
					return nil, errors.Wrapf(err, "could not apply attestation for validator index %d", validatorIndex)
				}

				if slashing == nil {
					continue
				}

				root, err := slashing.HashTreeRoot()
				if err != nil {
					return nil, errors.Wrap(err, "could not hash tree root for attester slashing")
				}

				slashings[root] = slashing
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
		chunk, err = s.getChunkFromDatabase(ctx, chunkKind, validatorChunkIndex, chunkIndex)
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
			chunk, err = s.getChunkFromDatabase(ctx, chunkKind, validatorChunkIndex, chunkIndex)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get chunk at index %d", chunkIndex)
			}
		}

		keepGoing, err := chunk.Update(
			chunkIndex,
			currentEpoch,
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

// Retrieve a chunk from database.
func (s *Service) getChunkFromDatabase(
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
	chunkIndexes []uint64,
) (map[uint64]Chunker, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.loadChunks")
	defer span.End()

	chunkKeys := make([][]byte, 0, len(chunkIndexes))
	for _, chunkIndex := range chunkIndexes {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(validatorChunkIndex, chunkIndex))
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

		chunksByChunkIdx[chunkIndexes[i]] = chunk
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
