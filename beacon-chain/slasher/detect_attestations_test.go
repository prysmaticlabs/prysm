package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_loadChunk_MinSpan(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)

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
	received, err := s.loadChunk(updatedChunks, 0, chunkIdx, kind)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	chunkIdx = uint64(2)
	received, err = s.loadChunk(updatedChunks, 0, chunkIdx, kind)
	require.NoError(t, err)
	require.DeepEqual(t, emptyChunk, received)

	// Save chunks to disk, the load them properly from disk.
	updatedChunks = map[uint64]Chunker{
		2: existingChunk,
		4: existingChunk,
		6: existingChunk,
	}
	err = s.saveUpdatedChunks(
		context.Background(),
		kind,
		updatedChunks,
		0,
	)
	require.NoError(t, err)
	// Check if the retrieved chunks match what we just saved to disk.
	for chunkIdx = range updatedChunks {
		input := make(map[uint64]Chunker)
		received, err := s.loadChunk(input, 0, chunkIdx, kind)
		require.NoError(t, err)
		require.DeepEqual(t, existingChunk, received)
	}
}

func TestService_loadChunk_MaxSpan(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)

	// Check if the chunk at chunk index already exists in-memory.
	params := DefaultParams()
	kind := slashertypes.MaxSpan
	emptyChunk := EmptyMaxSpanChunksSlice(params)
	existingChunk := EmptyMaxSpanChunksSlice(params)
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
	received, err := s.loadChunk(updatedChunks, 0, chunkIdx, kind)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	chunkIdx = uint64(2)
	received, err = s.loadChunk(updatedChunks, 0, chunkIdx, kind)
	require.NoError(t, err)
	require.DeepEqual(t, emptyChunk, received)

	// Save chunks to disk, the load them properly from disk.
	updatedChunks = map[uint64]Chunker{
		2: existingChunk,
		4: existingChunk,
		6: existingChunk,
	}
	err = s.saveUpdatedChunks(
		context.Background(),
		kind,
		updatedChunks,
		0,
	)
	require.NoError(t, err)
	// Check if the retrieved chunks match what we just saved to disk.
	for chunkIdx = range updatedChunks {
		input := make(map[uint64]Chunker)
		received, err := s.loadChunk(input, 0, chunkIdx, kind)
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
