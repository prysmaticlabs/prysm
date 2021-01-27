package slasher

import (
	"context"
	"fmt"
	"math"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
)

// Chunker defines a struct which implements a chunk for a min/max span for a validator
// used for surround vote detection in slasher. The interface defines methods used to check
// if an attestation is slashable for a validator index based on the contents of
// the chunk as well as the ability to update the data in the chunk with incoming information.
// For detailed information on what a chunk is and how it works, refer to our design document:
// https://hackmd.io/@Yl0VNGYRR6aeDrHHQNhuaA/prysm-slasher.
type Chunker interface {
	NeutralElement() uint16
	Chunk() []uint16
	CheckSlashable(
		ctx context.Context,
		slasherDB db.Database,
		validatorIdx types.ValidatorIndex,
		attestation *ethpb.IndexedAttestation,
	) (bool, slashertypes.SlashingKind, error)
	Update(
		chunkIdx uint64,
		validatorIdx types.ValidatorIndex,
		startEpoch,
		currentEpoch,
		newTargetEpoch types.Epoch,
	) (bool, error)
}

// MinSpanChunk represents a validator min span where for a given epoch, e, and attestations
// a validator index has produced, atts, such that min_spans[e] is defined as
// min((att.target.epoch - e) for att in attestations) where att.source.epoch > e.
// That is, it is the minimum distance between the specified epoch and all attestation
// target epochs a validator has created where att.source.epoch > e.
//
// Under ideal network conditions, where every target epoch immediately follows its source,
// min spans for validators will look as follows:
//
//  min_spans = [2, 2, 2, ..., 2]
//
// For more details on how these values are computed, see:
// https://hackmd.io/@Yl0VNGYRR6aeDrHHQNhuaA/prysm-slasher.
type MinSpanChunk struct {
	params *Parameters
	data   []uint16
}

// EmptyMinChunk initializes a min span chunk of length C*K for
// C = chunkSize and K = validatorChunkSize filled with neutral elements.
// For min spans, the neutral element is `undefined`, represented by MaxUint16.
func EmptyMinSpanChunk(config *Parameters) *MinSpanChunk {
	m := &MinSpanChunk{
		params: config,
	}
	data := make([]uint16, config.chunkSize*config.validatorChunkSize)
	for i := 0; i < len(data); i++ {
		data[i] = m.NeutralElement()
	}
	m.data = data
	return m
}

// EmptyMinChunkFrom initializes a min span chunk from a slice of uint16 values.
// Returns an error if the slice is not of length C*K for C = chunkSize and
// K = validatorChunkSize.
func MinChunkSpanFrom(config *Parameters, chunk []uint16) (*MinSpanChunk, error) {
	requiredLen := config.chunkSize * config.validatorChunkSize
	if uint64(len(chunk)) != requiredLen {
		return nil, fmt.Errorf("chunk has wrong length, %d, expected %d", len(chunk), requiredLen)
	}
	return &MinSpanChunk{
		params: config,
		data:   chunk,
	}, nil
}

// NeutralElement for a min span chunk is undefined, in this case
// using MaxUint16 as a sane value given it is impossible we reach it.
func (m *MinSpanChunk) NeutralElement() uint16 {
	return math.MaxUint16
}

// Chunk returns the underlying slice of uint16's for a validator min span.
func (m *MinSpanChunk) Chunk() []uint16 {
	return m.data
}

// CheckSlashable takes in a validator index and an incoming attestation
// and checks if the validator is slashable depending on the data
// within the chunk. Recall that for an incoming attestation, B, and an
// existing attestation, A:
//
//  B surrounds A if and only if B.target > min_spans[B.source]
//
// That is, this condition is sufficient to check if an incoming attestation
// is surrounding a previous one. We also check if we indeed have an existing
// attestation record in the database if the condition holds true in order
// to be confident of a slashable offense.
func (m *MinSpanChunk) CheckSlashable(
	ctx context.Context,
	slasherDB db.Database,
	validatorIdx types.ValidatorIndex,
	attestation *ethpb.IndexedAttestation,
) (bool, slashertypes.SlashingKind, error) {
	sourceEpoch := types.Epoch(attestation.Data.Source.Epoch)
	targetEpoch := types.Epoch(attestation.Data.Target.Epoch)
	minTarget, err := chunkDataAtEpoch(m.params, m.data, validatorIdx, sourceEpoch)
	if err != nil {
		return false, slashertypes.NotSlashable, errors.Wrapf(
			err, "could not get min target for validator %d at epoch %d", validatorIdx, sourceEpoch,
		)
	}
	if targetEpoch > minTarget {
		existingAttRecord, err := slasherDB.AttestationRecordForValidator(ctx, validatorIdx, minTarget)
		if err != nil {
			return false, slashertypes.NotSlashable, err
		}
		if existingAttRecord != nil {
			if sourceEpoch < types.Epoch(existingAttRecord.Source) {
				return true, slashertypes.SurroundingVote, nil
			}
		}
	}
	return false, slashertypes.NotSlashable, nil
}

// Update a min span chunk for a validator index starting at the current epoch, e_c, then updating
// down to e_c - H where H is the historyLength we keep for each span. This historyLength
// corresponds to the weak subjectivity period of eth2, see:
// https://github.com/ethereum/eth2.0-specs/blob/dev/specs/phase0/weak-subjectivity.md.
// This means our updates are done in a sliding window manner. For example, if the current epoch
// is 20 and the historyLength is 12, then we will update every value for the validator's min span
// from epoch 20 down to epoch 8 (since 20 - 12 = 8).
//
// Recall that for an epoch, e, min((att.target - e) for att in attestations where att.source > e)
// That is, it is the minimum distance between the specified epoch and all attestation
// target epochs a validator has created where att.source.epoch > e.
//
// Let's take a look at how this update will look for a real set of min span chunk:
// For the purposes of a simple example, let's set H = 12, meaning a min span
// will hold 12 epochs worth of attesting history. Then we set C = 4 meaning we will
// chunk the min span into arrays each of length 4.
//
//  validator_0_min_span = [2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2]
//
// After chunking into chunks of length C = 4:
//
//  validator_0_min_span_chunks = [[2, 2, 2, 2], [2, 2, 2, 2], [2, 2, 2, 2]]
//
// So assume we get an epoch 18, then, we need to update every epoch in the span from
// 18 down to 6. First, we find out which chunk epoch 18 falls into, which is calculated as:
// chunk_idx = (epoch % H) / C = (18 % 12) / 4 = (6 / 4) = 1.
//
//              epoch 18, chunk index 1
//                        |
//  [[2, 2, 2, 2], [2, 2, 2, 2], [2, 2, 2, 2]]
//
// Next up, we proceed with the update process, starting epoch 18
// al the way down to epoch 6. We will need to go down the array once
// and then wrap around at the end back down to epoch 6.
//
//      update every epoch from 18 down to 6
//                        |
//  <----------------------
//                        <-----------------<-
//                        |
//                      epoch 6
//                        |
//  [[2, 2, 2, 2], [2, 2, 2, 2], [2, 2, 2, 2]]
//
// Once we finish updating a chunk, we need to move on to the next chunk. This function
// returns a boolean named keepGoing which allows the caller to determine if we should
// continue and update the min chunk. We stop whenever we reach the min epoch we need
// to update, in our example, we stop at 6.
func (m *MinSpanChunk) Update(
	chunkIdx uint64,
	validatorIdx types.ValidatorIndex,
	startEpoch,
	currentEpoch,
	newTargetEpoch types.Epoch,
) (bool, error) {
	// The lowest epoch we need to update. This is a sliding window from (current epoch - H) where
	// H is the history length a min span for a validator stores.
	var minEpoch types.Epoch
	if uint64(currentEpoch) > (m.params.historyLength - 1) {
		minEpoch = currentEpoch.Sub(m.params.historyLength - 1)
	}
	epochInChunk := startEpoch
	// We go down the chunk, updating every value starting at start_epoch down to min_epoch.
	// As long as the epoch, e, in the same chunk index and e >= min_epoch, we proceed with
	// a for loop.
	for m.params.chunkIndex(epochInChunk) == chunkIdx && epochInChunk >= minEpoch {
		chunkTarget, err := chunkDataAtEpoch(m.params, m.data, validatorIdx, epochInChunk)
		if err != nil {
			return false, err
		}
		// If the newly incoming value is < the existing value, we update
		// the data in the min span to meet with its definition.
		if newTargetEpoch < chunkTarget {
			if err = setChunkDataAtEpoch(m.params, m.data, validatorIdx, epochInChunk, newTargetEpoch); err != nil {
				return false, err
			}
		} else {
			// We can stop because spans are guaranteed to be minimums and
			// if we did not meet the minimum condition, there is nothing to update.
			return false, nil
		}
		// We decrease our epoch index variable as long as it is > 0.
		if epochInChunk > 0 {
			epochInChunk -= 1
		}
	}
	// We should keep going and update the previous chunk if we are yet to reach
	// the minimum epoch required for the update procedure.
	keepGoing := epochInChunk >= minEpoch
	return keepGoing, nil
}

// Given a validator index and epoch, retrieves the target epoch at its specific
// index for the validator index + epoch pair in a min/max span chunk.
func chunkDataAtEpoch(
	params *Parameters, chunk []uint16, validatorIdx types.ValidatorIndex, epoch types.Epoch,
) (types.Epoch, error) {
	requiredLen := params.chunkSize * params.validatorChunkSize
	if uint64(len(chunk)) != requiredLen {
		return 0, fmt.Errorf("chunk has wrong length, %d, expected %d", len(chunk), requiredLen)
	}
	cellIdx := params.cellIndex(validatorIdx, epoch)
	if cellIdx >= uint64(len(chunk)) {
		return 0, fmt.Errorf("cell index %d out of bounds (len(chunk) = %d)", cellIdx, len(chunk))
	}
	distance := chunk[cellIdx]
	return epoch.Add(uint64(distance)), nil
}

// Updates the value at a specific index in a chunk for a validator index + epoch
// pair to a specified distance. Recall that for min spans, each element in a chunk
// is the minimum distance between the a given epoch, e, and all attestation target epochs
// a validator has created where att.source.epoch > e.
func setChunkDataAtEpoch(
	config *Parameters,
	chunk []uint16,
	validatorIdx types.ValidatorIndex,
	epochInChunk types.Epoch,
	targetEpoch types.Epoch,
) error {
	distance := epochDistance(targetEpoch, epochInChunk)
	cellIdx := config.cellIndex(validatorIdx, epochInChunk)
	if cellIdx >= uint64(len(chunk)) {
		return fmt.Errorf("cell index %d out of bounds (len(chunk) = %d)", cellIdx, len(chunk))
	}
	chunk[cellIdx] = distance
	return nil
}

// Computes a distance between two epochs. Given the result stored in
// min/max spans is maximum WEAK_SUBJECTIVITY_PERIOD, we are guaranteed the
// distance can be represented as a uint16 safely.
func epochDistance(epoch, baseEpoch types.Epoch) uint16 {
	return uint16(epoch.Sub(uint64(baseEpoch)))
}
