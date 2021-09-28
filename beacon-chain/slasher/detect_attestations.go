package slasher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// Takes in a list of indexed attestation wrappers and returns any
// found attester slashings to the caller.
func (s *Service) checkSlashableAttestations(
	ctx context.Context, atts []*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	currentEpoch := slots.EpochsSinceGenesis(s.genesisTime)
	slashings := make([]*ethpb.AttesterSlashing, 0)
	indices := make([]types.ValidatorIndex, 0)

	// TODO(#8331): Consider using goroutines and wait groups here.
	groupedAtts := s.groupByValidatorChunkIndex(atts)
	for validatorChunkIdx, batch := range groupedAtts {
		attSlashings, err := s.detectAllAttesterSlashings(ctx, &chunkUpdateArgs{
			validatorChunkIndex: validatorChunkIdx,
			currentEpoch:        currentEpoch,
		}, batch)

		slashings = append(slashings, attSlashings...)
		if err != nil {
			return nil, errors.Wrap(err, "Could not detect slashable attestations")
		}
		indices = append(indices, s.params.validatorIndicesInChunk(validatorChunkIdx)...)
	}
	if err := s.serviceCfg.Database.SaveLastEpochWrittenForValidators(ctx, indices, currentEpoch); err != nil {
		return nil, err
	}
	return slashings, nil
}

// Given a list of attestations all corresponding to a validator chunk index as well
// as the current epoch in time, we perform slashing detection.
// The process is as follows given a list of attestations:
//
// 1. Check for attester double votes using the list of attestations.
// 2. Group the attestations by chunk index.
// 3. Update the min and max spans for those grouped attestations, check if any slashings are
//    found in the process
// 4. Update the latest written epoch for all validators involved to the current epoch.
//
// This function performs a lot of critical actions and is split into smaller helpers for cleanliness.
func (s *Service) detectAllAttesterSlashings(
	ctx context.Context,
	args *chunkUpdateArgs,
	attestations []*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	// Check for double votes.
	doubleVoteSlashings, err := s.checkDoubleVotes(ctx, attestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not check slashable double votes")
	}

	// Group attestations by chunk index.
	groupedAtts := s.groupByChunkIndex(attestations)

	// Update min and max spans and retrieve any detected slashable offenses.
	surroundingSlashings, err := s.updateSpans(ctx, &chunkUpdateArgs{
		kind:                slashertypes.MinSpan,
		validatorChunkIndex: args.validatorChunkIndex,
		currentEpoch:        args.currentEpoch,
	}, groupedAtts)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not update min attestation spans for validator chunk index %d",
			args.validatorChunkIndex,
		)
	}

	surroundedSlashings, err := s.updateSpans(ctx, &chunkUpdateArgs{
		kind:                slashertypes.MaxSpan,
		validatorChunkIndex: args.validatorChunkIndex,
		currentEpoch:        args.currentEpoch,
	}, groupedAtts)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not update max attestation spans for validator chunk index %d",
			args.validatorChunkIndex,
		)
	}

	// Consolidate all slashings into a slice.
	slashings := make([]*ethpb.AttesterSlashing, 0)
	slashings = append(slashings, doubleVoteSlashings...)
	slashings = append(slashings, surroundingSlashings...)
	slashings = append(slashings, surroundedSlashings...)
	if len(slashings) > 0 {
		log.WithField("numSlashings", len(slashings)).Info("Slashable attestation offenses found")
	}
	return slashings, nil
}

// Check for attester slashing double votes by looking at every single validator index
// in each attestation's attesting indices and checking if there already exist records for such
// attestation's target epoch. If so, we append a double vote slashing object to a list of slashings
// we return to the caller.
func (s *Service) checkDoubleVotes(
	ctx context.Context, attestations []*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.checkDoubleVotes")
	defer span.End()
	// We check if there are any slashable double votes in the input list
	// of attestations with respect to each other.
	slashings := make([]*ethpb.AttesterSlashing, 0)
	existingAtts := make(map[string]*slashertypes.IndexedAttestationWrapper)
	for _, att := range attestations {
		for _, valIdx := range att.IndexedAttestation.AttestingIndices {
			key := uintToString(uint64(att.IndexedAttestation.Data.Target.Epoch)) + ":" + uintToString(valIdx)
			existingAtt, ok := existingAtts[key]
			if !ok {
				existingAtts[key] = att
				continue
			}
			if att.SigningRoot != existingAtt.SigningRoot {
				doubleVotesTotal.Inc()
				slashings = append(slashings, &ethpb.AttesterSlashing{
					Attestation_1: existingAtt.IndexedAttestation,
					Attestation_2: att.IndexedAttestation,
				})
			}
		}
	}

	// We check if there are any slashable double votes in the input list
	// of attestations with respect to our database.
	moreSlashings, err := s.checkDoubleVotesOnDisk(ctx, attestations)
	if err != nil {
		return nil, errors.Wrap(err, "could not check attestation double votes on disk")
	}
	return append(slashings, moreSlashings...), nil
}

// Check for double votes in our database given a list of incoming attestations.
func (s *Service) checkDoubleVotesOnDisk(
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
	attestationsByChunkIdx map[uint64][]*slashertypes.IndexedAttestationWrapper,
) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.updateSpans")
	defer span.End()
	// Determine the chunk indices we need to use for slashing detection.
	validatorIndices := s.params.validatorIndicesInChunk(args.validatorChunkIndex)
	chunkIndices, err := s.determineChunksToUpdateForValidators(ctx, args, validatorIndices)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not determine chunks to update for validator indices %v",
			validatorIndices,
		)
	}
	// Load the required chunks from disk.
	chunksByChunkIdx, err := s.loadChunks(ctx, args, chunkIndices)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not load chunks for chunk indices %v",
			chunkIndices,
		)
	}

	// Apply the attestations to the related chunks and find any
	// slashings along the way.
	slashings := make([]*ethpb.AttesterSlashing, 0)
	for _, attestationBatch := range attestationsByChunkIdx {
		for _, att := range attestationBatch {
			for _, validatorIdx := range att.IndexedAttestation.AttestingIndices {
				validatorIndex := types.ValidatorIndex(validatorIdx)
				computedValidatorChunkIdx := s.params.validatorChunkIndex(validatorIndex)

				// Every validator chunk index represents a range of validators.
				// If it possible that the validator index in this loop iteration is
				// not part of the validator chunk index we are updating chunks for.
				//
				// For example, if there are 4 validators per validator chunk index,
				// then validator chunk index 0 contains validator indices [0, 1, 2, 3]
				// If we see an attestation with attesting indices [3, 4, 5] and we are updating
				// chunks for validator chunk index 0, only validator index 3 should make
				// it past this line.
				if args.validatorChunkIndex != computedValidatorChunkIdx {
					continue
				}
				slashing, err := s.applyAttestationForValidator(
					ctx,
					args,
					validatorIndex,
					chunksByChunkIdx,
					att,
				)
				if err != nil {
					return nil, errors.Wrapf(
						err,
						"could not apply attestation for validator index %d",
						validatorIndex,
					)
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
	ctx, span := trace.StartSpan(ctx, "Slasher.determineChunksToUpdateForValidators")
	defer span.End()
	lastCurrentEpochs, err := s.serviceCfg.Database.LastEpochWrittenForValidators(ctx, validatorIndices)
	if err != nil {
		err = errors.Wrap(err, "could not get latest epoch attested for validators")
		return
	}

	// Initialize the last epoch written for each validator to 0.
	lastCurrentEpochByValidator := make(map[types.ValidatorIndex]types.Epoch, len(validatorIndices))
	for _, valIdx := range validatorIndices {
		lastCurrentEpochByValidator[valIdx] = 0
	}
	for _, lastEpoch := range lastCurrentEpochs {
		lastCurrentEpochByValidator[lastEpoch.ValidatorIndex] = lastEpoch.Epoch
	}

	// For every single validator and their last written current epoch, we determine
	// the chunk indices we need to update based on all the chunks between the last
	// epoch written and the current epoch, inclusive.
	chunkIndicesToUpdate := make(map[uint64]bool)

	for _, epoch := range lastCurrentEpochByValidator {
		latestEpochWritten := epoch
		for latestEpochWritten <= args.currentEpoch {
			chunkIdx := s.params.chunkIndex(latestEpochWritten)
			chunkIndicesToUpdate[chunkIdx] = true
			latestEpochWritten++
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
	validatorIndex types.ValidatorIndex,
	chunksByChunkIdx map[uint64]Chunker,
	attestation *slashertypes.IndexedAttestationWrapper,
) (*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.applyAttestationForValidator")
	defer span.End()
	sourceEpoch := attestation.IndexedAttestation.Data.Source.Epoch
	targetEpoch := attestation.IndexedAttestation.Data.Target.Epoch

	attestationDistance.Observe(float64(targetEpoch) - float64(sourceEpoch))

	chunkIdx := s.params.chunkIndex(sourceEpoch)
	chunk, err := s.getChunk(ctx, args, chunksByChunkIdx, chunkIdx)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get chunk at index %d", chunkIdx)
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
		chunk, err := s.getChunk(ctx, args, chunksByChunkIdx, chunkIdx)
		if err != nil {
			return nil, errors.Wrapf(err, "could not get chunk at index %d", chunkIdx)
		}
		keepGoing, err := chunk.Update(
			&chunkUpdateArgs{
				chunkIndex:   chunkIdx,
				currentEpoch: args.currentEpoch,
			},
			validatorIndex,
			startEpoch,
			targetEpoch,
		)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"could not update chunk at chunk index %d for validator index %d and current epoch %d",
				chunkIdx,
				validatorIndex,
				args.currentEpoch,
			)
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

// Retrieves a chunk at a chunk index from a map. If such chunk does not exist, which
// should be rare (occurring when we receive an attestation with source and target epochs
// that span multiple chunk indices), then we fallback to fetching from disk.
func (s *Service) getChunk(
	ctx context.Context,
	args *chunkUpdateArgs,
	chunksByChunkIdx map[uint64]Chunker,
	chunkIdx uint64,
) (Chunker, error) {
	chunk, ok := chunksByChunkIdx[chunkIdx]
	if ok {
		return chunk, nil
	}
	// We can ensure we load the appropriate chunk we need by fetching from the DB.
	diskChunks, err := s.loadChunks(ctx, args, []uint64{chunkIdx})
	if err != nil {
		return nil, errors.Wrapf(err, "could not load chunk at index %d", chunkIdx)
	}
	if chunk, ok := diskChunks[chunkIdx]; ok {
		return chunk, nil
	}
	return nil, fmt.Errorf("could not retrieve chunk at chunk index %d from disk", chunkIdx)
}

// Load chunks for a specified list of chunk indices. We attempt to load it from the database.
// If the data exists, then we initialize a chunk of a specified kind. Otherwise, we create
// an empty chunk, add it to our map, and then return it to the caller.
func (s *Service) loadChunks(
	ctx context.Context,
	args *chunkUpdateArgs,
	chunkIndices []uint64,
) (map[uint64]Chunker, error) {
	ctx, span := trace.StartSpan(ctx, "Slasher.loadChunks")
	defer span.End()
	chunkKeys := make([][]byte, 0, len(chunkIndices))
	for _, chunkIdx := range chunkIndices {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(args.validatorChunkIndex, chunkIdx))
	}
	rawChunks, chunksExist, err := s.serviceCfg.Database.LoadSlasherChunks(ctx, args.kind, chunkKeys)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"could not load slasher chunk index",
		)
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
			return nil, errors.Wrap(err, "could not initialize chunk")
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
	ctx, span := trace.StartSpan(ctx, "Slasher.saveUpdatedChunks")
	defer span.End()
	chunkKeys := make([][]byte, 0, len(updatedChunksByChunkIdx))
	chunks := make([][]uint16, 0, len(updatedChunksByChunkIdx))
	for chunkIdx, chunk := range updatedChunksByChunkIdx {
		chunkKeys = append(chunkKeys, s.params.flatSliceID(args.validatorChunkIndex, chunkIdx))
		chunks = append(chunks, chunk.Chunk())
	}
	chunksSavedTotal.Add(float64(len(chunks)))
	return s.serviceCfg.Database.SaveSlasherChunks(ctx, args.kind, chunkKeys, chunks)
}
