package peers_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPeerScorer_BlockProvider_Score(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerConfig{
			BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
				ProcessedBatchWeight: 0.05,
			},
		},
	})
	scorer := peerStatuses.Scorers().BlockProviderScorer()
	batchSize := uint64(flags.Get().BlockBatchLimit)

	// Start with non-exitent provider.
	assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for unregistered provider")
	// Increment to zero (provider is added to cache but score is unchanged).
	scorer.IncrementProcessedBlocks("peer1", 0)
	assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for registered provider")

	// Partial score (less than a single batch of blocks processed).
	scorer.IncrementProcessedBlocks("peer1", batchSize/2)
	assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score")

	// Single batch.
	scorer.IncrementProcessedBlocks("peer1", batchSize)
	assert.Equal(t, roundScore(0.05), scorer.Score("peer1"), "Unexpected score")

	// Multiple batches.
	scorer.IncrementProcessedBlocks("peer2", batchSize*13)
	assert.Equal(t, roundScore(0.05*13), scorer.Score("peer2"), "Unexpected score")
}

func TestPeerScorer_BlockProvider_GettersSetters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		ScorerParams: &peers.PeerScorerConfig{},
	})
	scorer := peerStatuses.Scorers().BlockProviderScorer()

	assert.Equal(t, uint64(0), scorer.ProcessedBlocks("peer1"), "Unexpected count for unregistered peer")
	scorer.IncrementProcessedBlocks("peer1", 64)
	assert.Equal(t, uint64(64), scorer.ProcessedBlocks("peer1"))
}

func TestPeerScorer_BlockProvider_Sorted(t *testing.T) {
	batchSize := uint64(flags.Get().BlockBatchLimit)
	tests := []struct {
		name   string
		update func(s *peers.BlockProviderScorer)
		have   []peer.ID
		want   []peer.ID
	}{
		{
			name:   "no peers",
			update: func(s *peers.BlockProviderScorer) {},
			have:   []peer.ID{},
			want:   []peer.ID{},
		},
		{
			name: "same scores",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", 16)
				s.IncrementProcessedBlocks("peer2", 16)
				s.IncrementProcessedBlocks("peer3", 16)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer2", "peer3"},
		},
		{
			name: "same scores multiple batches",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*7+16)
				s.IncrementProcessedBlocks("peer2", batchSize*7+16)
				s.IncrementProcessedBlocks("peer3", batchSize*7+16)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer2", "peer3"},
		},
		{
			name: "same scores multiple batches unequal blocks",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*7+6)
				s.IncrementProcessedBlocks("peer2", batchSize*7+16)
				s.IncrementProcessedBlocks("peer3", batchSize*7+26)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer2", "peer3"},
		},
		{
			name: "different scores",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*3)
				s.IncrementProcessedBlocks("peer2", batchSize*1)
				s.IncrementProcessedBlocks("peer3", batchSize*2)
			},
			have: []peer.ID{"peer3", "peer2", "peer1"},
			want: []peer.ID{"peer1", "peer3", "peer2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
						ProcessedBatchWeight: 0.2,
					},
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			tt.update(scorer)
			assert.DeepEqual(t, tt.want, scorer.Sorted(tt.have))
		})
	}
}

func TestPeerScorer_BlockProvider_MaxScore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	batchSize := uint64(flags.Get().BlockBatchLimit)

	tests := []struct {
		name   string
		update func(s *peers.BlockProviderScorer)
		want   float64
	}{
		{
			// Minimal max.score is a reward for a single batch.
			name:   "no peers",
			update: nil,
			want:   0.2,
		},
		{
			name: "partial batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize/4)
			},
			want: 0.2,
		},
		{
			name: "single batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize)
			},
			want: 0.2,
		},
		{
			name: "3/2 of a batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*3/2)
			},
			want: 0.2,
		},
		{
			name: "multiple batches",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*5)
			},
			want: 0.2 * 5,
		},
		{
			name: "multiple peers",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*5)
				s.IncrementProcessedBlocks("peer1", batchSize)
				s.IncrementProcessedBlocks("peer2", batchSize*10)
				s.IncrementProcessedBlocks("peer1", batchSize/4)
			},
			want: 0.2 * 10,
		},
		{
			// Even after stats is decayed, max. attained blocks count must remain
			// (as a ballpark of overall performance of peers during life-cycle of service).
			name: "decaying",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*5)
				s.IncrementProcessedBlocks("peer1", batchSize)
				s.IncrementProcessedBlocks("peer2", batchSize*10)
				s.IncrementProcessedBlocks("peer1", batchSize/4)
				for i := 0; i < 10; i++ {
					s.Decay()
				}
				assert.Equal(t, uint64(0), s.ProcessedBlocks("peer1"))
				assert.Equal(t, uint64(0), s.ProcessedBlocks("peer2"))
			},
			want: 0.2 * 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
						ProcessedBatchWeight: 0.2,
					},
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			if tt.update != nil {
				tt.update(scorer)
			}
			assert.Equal(t, tt.want, scorer.MaxScore())
		})
	}
}
