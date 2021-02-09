package slasher

import (
	"context"
	"fmt"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_applyCurrentEpochToValidators(t *testing.T) {
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
	// Given the chunk size is 2 epochs per chunk, updating with current epoch == 3
	// will mean that we are updating chunk index 0 and 1 worth of data.
	updatedChunks, err := s.applyCurrentEpochToValidators(
		ctx,
		&chunkUpdateOptions{
			currentEpoch: 3,
		},
		validators,
	)
	require.NoError(t, err)
	require.Equal(t, 2, len(updatedChunks))
	chunk0, ok := updatedChunks[0]
	require.Equal(t, true, ok)
	chunk1, ok := updatedChunks[1]
	require.Equal(t, true, ok)
	fmt.Println(chunk0, chunk1)
}

func TestService_loadChunk_MinSpan(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	kind := slashertypes.MinSpan
	params := DefaultParams()
	existingChunk := EmptyMinSpanChunksSlice(params)
	emptyChunk := EmptyMinSpanChunksSlice(params)
	validatorIdx := types.ValidatorIndex(0)
	epochInChunk := types.Epoch(0)
	targetEpoch := types.Epoch(2)
	err := setChunkDataAtEpoch(
		params,
		existingChunk.Chunk(),
		validatorIdx,
		epochInChunk,
		targetEpoch,
	)
	require.NoError(t, err)
	require.DeepNotEqual(t, existingChunk, emptyChunk)
	chunkIdx := uint64(1)
	updatedChunks := map[uint64]Chunker{
		chunkIdx: existingChunk,
	}
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	received, err := s.loadChunk(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		chunkIndex:          chunkIdx,
		kind:                kind,
	}, updatedChunks)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	chunkIdx = uint64(2)
	received, err = s.loadChunk(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		chunkIndex:          chunkIdx,
		kind:                kind,
	}, updatedChunks)
	require.NoError(t, err)
	require.DeepEqual(t, emptyChunk, received)

	// Save chunks to disk, the load them properly from disk.
	updatedChunks = map[uint64]Chunker{
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
	for chunkIdx = range updatedChunks {
		input := make(map[uint64]Chunker)
		received, err = s.loadChunk(ctx, &chunkUpdateOptions{
			validatorChunkIndex: 0,
			chunkIndex:          chunkIdx,
			kind:                kind,
		}, input)
		require.NoError(t, err)
		require.DeepEqual(t, existingChunk, received)
	}
}

func TestService_loadChunk_MaxSpan(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)
	ctx := context.Background()

	// Check if the chunk at chunk index already exists in-memory.
	kind := slashertypes.MaxSpan
	params := DefaultParams()
	existingChunk := EmptyMaxSpanChunksSlice(params)
	emptyChunk := EmptyMaxSpanChunksSlice(params)
	validatorIdx := types.ValidatorIndex(0)
	epochInChunk := types.Epoch(0)
	targetEpoch := types.Epoch(2)
	err := setChunkDataAtEpoch(
		params,
		existingChunk.Chunk(),
		validatorIdx,
		epochInChunk,
		targetEpoch,
	)
	require.NoError(t, err)
	require.DeepNotEqual(t, existingChunk, emptyChunk)
	chunkIdx := uint64(1)
	updatedChunks := map[uint64]Chunker{
		chunkIdx: existingChunk,
	}
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
	}
	received, err := s.loadChunk(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		chunkIndex:          chunkIdx,
		kind:                kind,
	}, updatedChunks)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	chunkIdx = uint64(2)
	received, err = s.loadChunk(ctx, &chunkUpdateOptions{
		validatorChunkIndex: 0,
		chunkIndex:          chunkIdx,
		kind:                kind,
	}, updatedChunks)
	require.NoError(t, err)
	require.DeepEqual(t, emptyChunk, received)

	// Save chunks to disk, the load them properly from disk.
	updatedChunks = map[uint64]Chunker{
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
	for chunkIdx = range updatedChunks {
		input := make(map[uint64]Chunker)
		received, err = s.loadChunk(ctx, &chunkUpdateOptions{
			validatorChunkIndex: 0,
			chunkIndex:          chunkIdx,
			kind:                kind,
		}, input)
		require.NoError(t, err)
		require.DeepEqual(t, existingChunk, received)
	}
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
