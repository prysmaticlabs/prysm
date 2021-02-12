package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	logTest "github.com/sirupsen/logrus/hooks/test"

	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func Test_determineChunksToUpdateForValidators_FromLatestWrittenEpoch(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	s := &Service{
		params: &Parameters{
			chunkSize:          2, // 2 epochs in a chunk.
			validatorChunkSize: 2, // 2 validators in a chunk.
			historyLength:      4,
		},
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	validators := []types.ValidatorIndex{
		1, 2,
	}
	currentEpoch := types.Epoch(3)

	// Set the latest written epoch for validators to current epoch - 1.
	latestWrittenEpoch := currentEpoch - 1
	err := beaconDB.SaveLatestEpochAttestedForValidators(ctx, validators, latestWrittenEpoch)
	require.NoError(t, err)

	// Because the validators have no recorded latest epoch written in the database,
	// Because the latest written epoch for the input validators is == 2, we expect
	// that we will update all epochs from 2 up to 3 (the current epoch). This is all
	// safe contained in chunk index 1.
	chunkIndices, err := s.determineChunksToUpdateForValidators(
		ctx,
		&chunkUpdateOptions{
			currentEpoch: currentEpoch,
		},
		validators,
	)
	require.NoError(t, err)
	require.DeepEqual(t, []uint64{1}, chunkIndices)
}

func Test_determineChunksToUpdateForValidators_FromGenesis(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	s := &Service{
		params: &Parameters{
			chunkSize:          2, // 2 epochs in a chunk.
			validatorChunkSize: 2, // 2 validators in a chunk.
			historyLength:      4,
		},
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	validators := []types.ValidatorIndex{
		1, 2,
	}
	// Because the validators have no recorded latest epoch written in the database,
	// we expect that we will update all epochs from genesis up to the current epoch.
	// Given the chunk size is 2 epochs per chunk, updating with current epoch == 3
	// will mean that we should be updating from epoch 0 to 3, meaning chunk indices 0 and 1.
	chunkIndices, err := s.determineChunksToUpdateForValidators(
		ctx,
		&chunkUpdateOptions{
			currentEpoch: 3,
		},
		validators,
	)
	require.NoError(t, err)
	require.DeepEqual(t, []uint64{0, 1}, chunkIndices)
}

func Test_applyAttestationForValidator_MinSpanChunk(t *testing.T) {
	ctx := context.Background()
	beaconDB := dbtest.SetupDB(t)
	params := DefaultParams()
	srv := &Service{
		params: params,
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	// We initialize an empty chunks slice.
	chunk := EmptyMinSpanChunksSlice(params)
	chunkIdx := uint64(0)
	currentEpoch := types.Epoch(3)
	validatorIdx := types.ValidatorIndex(0)
	opts := &chunkUpdateOptions{
		chunkIndex:     chunkIdx,
		currentEpoch:   currentEpoch,
		validatorIndex: validatorIdx,
	}
	chunksByChunkIdx := map[uint64]Chunker{
		chunkIdx: chunk,
	}

	// Attestation with a different chunk index that is
	// not found in the input map should return nil.
	// when applying the attestation for the validator.
	unknownSource := types.Epoch(params.chunkSize + 1)
	unknownAtt := createAttestation(unknownSource, unknownSource+1)
	slashing, err := srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		unknownAtt,
	)
	require.NoError(t, err)
	require.Equal(t, true, slashing == nil)

	// We apply attestation with (source 1, target 2) for our validator.
	source := types.Epoch(1)
	target := types.Epoch(2)
	att := createAttestation(source, target)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		att,
	)
	require.NoError(t, err)
	require.Equal(t, true, slashing == nil)
	err = beaconDB.SaveAttestationRecordsForValidators(
		ctx,
		[]types.ValidatorIndex{validatorIdx},
		[]*slashertypes.CompactAttestation{att},
	)
	require.NoError(t, err)

	// Next, we apply an attestation with (source 0, target 3) and
	// expect a slashable offense to be returned.
	source = types.Epoch(0)
	target = types.Epoch(3)
	slashableAtt := createAttestation(source, target)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		slashableAtt,
	)
	require.NoError(t, err)
	require.NotNil(t, slashing)
	require.Equal(t, slashertypes.SurroundingVote, slashing.Kind)
}

func Test_applyAttestationForValidator_MaxSpanChunk(t *testing.T) {
	ctx := context.Background()
	beaconDB := dbtest.SetupDB(t)
	params := DefaultParams()
	srv := &Service{
		params: params,
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	// We initialize an empty chunks slice.
	chunk := EmptyMaxSpanChunksSlice(params)
	chunkIdx := uint64(0)
	currentEpoch := types.Epoch(3)
	validatorIdx := types.ValidatorIndex(0)
	opts := &chunkUpdateOptions{
		chunkIndex:     chunkIdx,
		currentEpoch:   currentEpoch,
		validatorIndex: validatorIdx,
	}
	chunksByChunkIdx := map[uint64]Chunker{
		chunkIdx: chunk,
	}

	// Attestation with a different chunk index that is
	// not found in the input map should return nil.
	// when applying the attestation for the validator.
	unknownSource := types.Epoch(params.chunkSize + 1)
	unknownAtt := createAttestation(unknownSource, unknownSource+1)
	slashing, err := srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		unknownAtt,
	)
	require.NoError(t, err)
	require.Equal(t, true, slashing == nil)

	// We apply attestation with (source 0, target 3) for our validator.
	source := types.Epoch(0)
	target := types.Epoch(3)
	att := createAttestation(source, target)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		att,
	)
	require.NoError(t, err)
	require.Equal(t, true, slashing == nil)
	err = beaconDB.SaveAttestationRecordsForValidators(
		ctx,
		[]types.ValidatorIndex{validatorIdx},
		[]*slashertypes.CompactAttestation{att},
	)
	require.NoError(t, err)

	// Next, we apply an attestation with (source 1, target 2) and
	// expect a slashable offense to be returned.
	source = types.Epoch(1)
	target = types.Epoch(2)
	slashableAtt := createAttestation(source, target)
	slashing, err = srv.applyAttestationForValidator(
		ctx,
		opts,
		chunksByChunkIdx,
		slashableAtt,
	)
	require.NoError(t, err)
	require.NotNil(t, slashing)
	require.Equal(t, slashertypes.SurroundedVote, slashing.Kind)
}

func Test_loadChunks_MinSpans(t *testing.T) {
	testLoadChunks(t, slashertypes.MinSpan)
}

func Test_loadChunks_MaxSpans(t *testing.T) {
	testLoadChunks(t, slashertypes.MaxSpan)
}

func testLoadChunks(t *testing.T, kind slashertypes.ChunkKind) {
	beaconDB := dbtest.SetupDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	params := DefaultParams()
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	var emptyChunk Chunker
	if kind == slashertypes.MinSpan {
		emptyChunk = EmptyMinSpanChunksSlice(params)
	} else {
		emptyChunk = EmptyMaxSpanChunksSlice(params)
	}
	chunkIdx := uint64(2)
	received, err := s.loadChunks(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		kind:                kind,
	}, []uint64{chunkIdx})
	require.NoError(t, err)
	wanted := map[uint64]Chunker{
		chunkIdx: emptyChunk,
	}
	require.DeepEqual(t, wanted, received)

	// Save chunks to disk, then load them properly from disk.
	var existingChunk Chunker
	if kind == slashertypes.MinSpan {
		existingChunk = EmptyMinSpanChunksSlice(params)
	} else {
		existingChunk = EmptyMaxSpanChunksSlice(params)
	}
	validatorIdx := types.ValidatorIndex(0)
	epochInChunk := types.Epoch(0)
	targetEpoch := types.Epoch(2)
	err = setChunkDataAtEpoch(
		params,
		existingChunk.Chunk(),
		validatorIdx,
		epochInChunk,
		targetEpoch,
	)
	require.NoError(t, err)
	require.DeepNotEqual(t, existingChunk, emptyChunk)

	updatedChunks := map[uint64]Chunker{
		2: existingChunk,
		4: existingChunk,
		6: existingChunk,
	}
	err = s.saveUpdatedChunks(
		ctx,
		&chunkUpdateOptions{
			validatorChunkIndex: 0,
			kind:                kind,
		},
		updatedChunks,
	)
	require.NoError(t, err)
	// Check if the retrieved chunks match what we just saved to disk.
	received, err = s.loadChunks(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		kind:                kind,
	}, []uint64{2, 4, 6})
	require.NoError(t, err)
	require.DeepEqual(t, updatedChunks, received)
}

func TestService_processQueuedAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
		attestationQueue: []*slashertypes.CompactAttestation{
			{
				AttestingIndices: []uint64{0, 1},
				Source:           0,
				Target:           1,
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan uint64)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedAttestations(ctx, tickerChan)
		exitChan <- struct{}{}
	}()

	// Send a value over the ticker.
	tickerChan <- 0
	cancel()
	<-exitChan
	assert.LogsContain(t, hook, "Epoch reached, processing queued")
}
