package slasher

import (
	"context"
	"reflect"
	"testing"

	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_groupByValidatorChunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		params *Parameters
		atts   []*slashertypes.CompactAttestation
		want   map[uint64][]*slashertypes.CompactAttestation
	}{
		{
			name:   "No attestations returns empty map",
			params: DefaultParams(),
			atts:   make([]*slashertypes.CompactAttestation, 0),
			want:   make(map[uint64][]*slashertypes.CompactAttestation),
		},
		{
			name: "Groups multiple attestations belonging to single validator chunk",
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*slashertypes.CompactAttestation{
				{
					AttestingIndices: []uint64{0, 1},
				},
				{
					AttestingIndices: []uint64{0, 1},
				},
			},
			want: map[uint64][]*slashertypes.CompactAttestation{
				0: {
					{
						AttestingIndices: []uint64{0, 1},
					},
					{
						AttestingIndices: []uint64{0, 1},
					},
				},
			},
		},
		{
			name: "Groups single attestation belonging to multiple validator chunk",
			params: &Parameters{
				validatorChunkSize: 2,
			},
			atts: []*slashertypes.CompactAttestation{
				{
					AttestingIndices: []uint64{0, 2, 4},
				},
			},
			want: map[uint64][]*slashertypes.CompactAttestation{
				0: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
				1: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
				2: {
					{
						AttestingIndices: []uint64{0, 2, 4},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params: tt.params,
			}
			if got := s.groupByValidatorChunkIndex(tt.atts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByValidatorChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_groupByChunkIndex(t *testing.T) {
	tests := []struct {
		name   string
		params *Parameters
		atts   []*slashertypes.CompactAttestation
		want   map[uint64][]*slashertypes.CompactAttestation
	}{
		{
			name:   "No attestations returns empty map",
			params: DefaultParams(),
			atts:   make([]*slashertypes.CompactAttestation, 0),
			want:   make(map[uint64][]*slashertypes.CompactAttestation),
		},
		{
			name: "Groups multiple attestations belonging to single chunk",
			params: &Parameters{
				chunkSize:     2,
				historyLength: 3,
			},
			atts: []*slashertypes.CompactAttestation{
				{
					Source: 0,
				},
				{
					Source: 1,
				},
			},
			want: map[uint64][]*slashertypes.CompactAttestation{
				0: {
					{
						Source: 0,
					},
					{
						Source: 1,
					},
				},
			},
		},
		{
			name: "Groups multiple attestations belonging to multiple chunks",
			params: &Parameters{
				chunkSize:     2,
				historyLength: 3,
			},
			atts: []*slashertypes.CompactAttestation{
				{
					Source: 0,
				},
				{
					Source: 1,
				},
				{
					Source: 2,
				},
			},
			want: map[uint64][]*slashertypes.CompactAttestation{
				0: {
					{
						Source: 0,
					},
					{
						Source: 1,
					},
				},
				1: {
					{
						Source: 2,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				params: tt.params,
			}
			if got := s.groupByChunkIndex(tt.atts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("groupByChunkIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_loadChunk(t *testing.T) {
	beaconDB := dbtest.SetupDB(t)

	// Check if the chunk at chunk index already exists in-memory.
	existingChunk := EmptyMinSpanChunksSlice(DefaultParams())
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
	received, err := s.loadChunk(updatedChunks, 0, chunkIdx, slashertypes.MinSpan)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// If a chunk at a chunk index does not exist, ensure it
	// is initialized as an empty chunk.
	chunkIdx = uint64(2)
	received, err = s.loadChunk(updatedChunks, 0, chunkIdx, slashertypes.MinSpan)
	require.NoError(t, err)
	require.DeepEqual(t, existingChunk, received)

	// Save chunks to disk, the load them properly from disk.
}

func TestService_processQueuedAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	s := &Service{
		params: DefaultParams(),
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
