package slasher

import (
	"context"
	"math"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

var _ = Chunker(&MinSpanChunksSlice{})

func TestMinSpanChunksSlice_Chunk(t *testing.T) {
	chunk := EmptyMinSpanChunksSlice(&Parameters{
		chunkSize:          2,
		validatorChunkSize: 2,
	})
	wanted := []uint16{math.MaxUint16, math.MaxUint16, math.MaxUint16, math.MaxUint16}
	require.Equal(t, wanted, chunk.Chunk())
}

func TestMinSpanChunksSlice_NeutralElement(t *testing.T) {
	chunk := EmptyMinSpanChunksSlice(&Parameters{})
	require.Equal(t, math.MaxUint16, chunk.NeutralElement())
}

func TestMinSpanChunksSlice_MinChunkSpanFrom(t *testing.T) {
	params := &Parameters{
		chunkSize:          3,
		validatorChunkSize: 2,
	}
	_, err := MinChunkSpansSliceFrom(params, []uint16{})
	require.ErrorContains(t, "chunk has wrong length", err)

	data := []uint16{2, 2, 2, 2, 2, 2}
	chunk, err := MinChunkSpansSliceFrom(&Parameters{
		chunkSize:          3,
		validatorChunkSize: 2,
	}, data)
	require.NoError(t, err)
	require.DeepEqual(t, data, chunk.Chunk())
}

func TestMinSpanChunksSlice_CheckSlashable(t *testing.T) {
	ctx := context.Background()
	beaconDB := dbtest.SetupDB(t)
	params := &Parameters{
		chunkSize:          3,
		validatorChunkSize: 2,
		historyLength:      3,
	}
	validatorIdx := types.ValidatorIndex(1)
	source := types.Epoch(1)
	target := types.Epoch(2)
	att := createAttestation(source, target)

	// A faulty chunk should lead to error.
	chunk := &MinSpanChunksSlice{
		params: params,
		data:   []uint16{},
	}
	_, _, err := chunk.CheckSlashable(ctx, nil, validatorIdx, att)
	require.ErrorContains(t, "could not get min target for validator", err)

	// We initialize a proper slice with 2 chunks with chunk size 3, 2 validators, and
	// a history length of 3 representing a perfect attesting history.
	//
	//     val0     val1
	//   {     }  {     }
	//  [2, 2, 2, 2, 2, 2]
	data := []uint16{2, 2, 2, 2, 2, 2}
	chunk, err = MinChunkSpansSliceFrom(params, data)
	require.NoError(t, err)

	// An attestation with source 1 and target 2 should not be slashable
	// based on our min chunk for either validator.
	slashable, kind, err := chunk.CheckSlashable(ctx, beaconDB, validatorIdx, att)
	require.NoError(t, err)
	require.Equal(t, false, slashable)
	require.Equal(t, slashertypes.NotSlashable, kind)

	slashable, kind, err = chunk.CheckSlashable(ctx, beaconDB, validatorIdx.Sub(1), att)
	require.NoError(t, err)
	require.Equal(t, false, slashable)
	require.Equal(t, slashertypes.NotSlashable, kind)

	// Next up we initialize an empty chunks slice and mark an attestation
	// with (source 1, target 2) as attested.
	chunk = EmptyMinSpanChunksSlice(params)
	source = types.Epoch(1)
	target = types.Epoch(2)
	att = createAttestation(source, target)
	chunkIdx := uint64(0)
	startEpoch := target
	currentEpoch := target
	_, err = chunk.Update(chunkIdx, validatorIdx, startEpoch, currentEpoch, target)
	require.NoError(t, err)

	// Next up, we create a surrounding vote, but it should NOT be slashable
	// because we have an existing attestation record in our database at the min target epoch.
	source = types.Epoch(0)
	target = types.Epoch(3)
	surroundingVote := createAttestation(source, target)

	slashable, kind, err = chunk.CheckSlashable(ctx, beaconDB, validatorIdx, surroundingVote)
	require.NoError(t, err)
	require.Equal(t, false, slashable)
	require.Equal(t, slashertypes.NotSlashable, kind)

	// Next up, we save the old attestation record, then check if the
	// surrounding vote is indeed slashable.
	err = beaconDB.SaveAttestationRecordForValidator(ctx, validatorIdx, [32]byte{1}, att)
	require.NoError(t, err)

	slashable, kind, err = chunk.CheckSlashable(ctx, beaconDB, validatorIdx, surroundingVote)
	require.NoError(t, err)
	require.Equal(t, true, slashable)
	require.Equal(t, slashertypes.SurroundingVote, kind)
}

func Test_chunkDataAtEpoch_SetRetrieve(t *testing.T) {
	// We initialize a chunks slice for 2 validators and with chunk size 3,
	// which will look as follows:
	//
	//     val0     val1
	//   {     }  {     }
	//  [2, 2, 2, 2, 2, 2]
	//
	// To give an example, epoch 1 for validator 1 will be at the following position:
	//
	//  [2, 2, 2, 2, 2, 2]
	//               |-> epoch 1, validator 1.
	params := &Parameters{
		chunkSize:          3,
		validatorChunkSize: 2,
	}
	chunk := []uint16{2, 2, 2, 2, 2, 2}
	validatorIdx := types.ValidatorIndex(1)
	epochInChunk := types.Epoch(1)

	// We expect a chunk with the wrong length to throw an error.
	_, err := chunkDataAtEpoch(params, []uint16{}, validatorIdx, epochInChunk)
	require.ErrorContains(t, "chunk has wrong length", err)

	// We update the value for epoch 1 using target epoch 6.
	targetEpoch := types.Epoch(6)
	err = setChunkDataAtEpoch(params, chunk, validatorIdx, epochInChunk, targetEpoch)
	require.NoError(t, err)
	// We expect the retrieved value at epoch 1 is the target epoch 6.
	received, err := chunkDataAtEpoch(params, chunk, validatorIdx, epochInChunk)
	require.NoError(t, err)
	assert.Equal(t, targetEpoch, received)
}

func createAttestation(source, target types.Epoch) *ethpb.IndexedAttestation {
	return &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: uint64(source),
			},
			Target: &ethpb.Checkpoint{
				Epoch: uint64(target),
			},
		},
	}
}
