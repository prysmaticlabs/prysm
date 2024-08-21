package cache

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v5/proto/engine/v1"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestSaveSignedExecutionPayloadHeader(t *testing.T) {
	resetHeaderCache()

	t.Run("First header should be added to cache", func(t *testing.T) {
		header := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		SaveSignedExecutionPayloadHeader(header)
		require.Equal(t, 1, len(cachedSignedExecutionPayloadHeader))
		require.Equal(t, header, cachedSignedExecutionPayloadHeader[1][0])
	})

	t.Run("Second header with higher slot should be added, and both slots should be in cache", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           100,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)
		require.Equal(t, 2, len(cachedSignedExecutionPayloadHeader))
		require.Equal(t, header1, cachedSignedExecutionPayloadHeader[1][0])
		require.Equal(t, header2, cachedSignedExecutionPayloadHeader[2][0])
	})

	t.Run("Third header with higher slot should replace the oldest slot", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           100,
			},
		}
		header3 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            3,
				ParentBlockHash: []byte("parent3"),
				Value:           100,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)
		SaveSignedExecutionPayloadHeader(header3)
		require.Equal(t, 2, len(cachedSignedExecutionPayloadHeader))
		require.Equal(t, header2, cachedSignedExecutionPayloadHeader[2][0])
		require.Equal(t, header3, cachedSignedExecutionPayloadHeader[3][0])
	})

	t.Run("Header with same slot but higher value should replace the existing one", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           200,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)
		require.Equal(t, 1, len(cachedSignedExecutionPayloadHeader[2]))
		require.Equal(t, header2, cachedSignedExecutionPayloadHeader[2][0])
	})

	t.Run("Header with different parent block hash should be appended to the same slot", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           200,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)
		require.Equal(t, 2, len(cachedSignedExecutionPayloadHeader[2]))
		require.Equal(t, header1, cachedSignedExecutionPayloadHeader[2][0])
		require.Equal(t, header2, cachedSignedExecutionPayloadHeader[2][1])
	})
}

func TestSignedExecutionPayloadHeader(t *testing.T) {
	cachedSignedExecutionPayloadHeader = make(map[primitives.Slot][]*enginev1.SignedExecutionPayloadHeader) // Reset global state before each test

	t.Run("Return header when slot and parentBlockHash match", func(t *testing.T) {
		header := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		SaveSignedExecutionPayloadHeader(header)
		result := SignedExecutionPayloadHeaderByHash(1, []byte("parent1"))
		require.NotNil(t, result)
		require.Equal(t, header, result)
	})

	t.Run("Return nil when no matching slot and parentBlockHash", func(t *testing.T) {
		header := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		SaveSignedExecutionPayloadHeader(header)
		result := SignedExecutionPayloadHeaderByHash(2, []byte("parent2"))
		require.IsNil(t, result)
	})

	t.Run("Return header when there are two slots in the cache and a match is found", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           200,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)

		// Check for the first header
		result1 := SignedExecutionPayloadHeaderByHash(1, []byte("parent1"))
		require.NotNil(t, result1)
		require.Equal(t, header1, result1)

		// Check for the second header
		result2 := SignedExecutionPayloadHeaderByHash(2, []byte("parent2"))
		require.NotNil(t, result2)
		require.Equal(t, header2, result2)
	})

	t.Run("Return nil when slot is evicted from cache", func(t *testing.T) {
		resetHeaderCache()
		header1 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            1,
				ParentBlockHash: []byte("parent1"),
				Value:           100,
			},
		}
		header2 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            2,
				ParentBlockHash: []byte("parent2"),
				Value:           200,
			},
		}
		header3 := &enginev1.SignedExecutionPayloadHeader{
			Message: &enginev1.ExecutionPayloadHeaderEPBS{
				Slot:            3,
				ParentBlockHash: []byte("parent3"),
				Value:           300,
			},
		}
		SaveSignedExecutionPayloadHeader(header1)
		SaveSignedExecutionPayloadHeader(header2)
		SaveSignedExecutionPayloadHeader(header3)

		// The first slot should be evicted, so result should be nil
		result := SignedExecutionPayloadHeaderByHash(1, []byte("parent1"))
		require.IsNil(t, result)

		// The second slot should still be present
		result = SignedExecutionPayloadHeaderByHash(2, []byte("parent2"))
		require.NotNil(t, result)
		require.Equal(t, header2, result)

		// The third slot should be present
		result = SignedExecutionPayloadHeaderByHash(3, []byte("parent3"))
		require.NotNil(t, result)
		require.Equal(t, header3, result)
	})
}

func resetHeaderCache() {
	cachedSignedExecutionPayloadHeader = make(map[primitives.Slot][]*enginev1.SignedExecutionPayloadHeader)
}
