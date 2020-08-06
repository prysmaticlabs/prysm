package peers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPeerScorer_BlockProvider_Score(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	batchSize := uint64(flags.Get().BlockBatchLimit)
	tests := []struct {
		name   string
		update func(scorer *peers.BlockProviderScorer)
		check  func(scorer *peers.BlockProviderScorer)
	}{
		{
			name: "nonexistent peer",
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for unregistered provider")
			},
		},
		{
			name: "increment with 0 score",
			update: func(scorer *peers.BlockProviderScorer) {
				// Increment to zero (provider is added to cache but score is unchanged).
				scorer.IncrementProcessedBlocks("peer1", 0)
			},
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for registered provider")
			},
		},
		{
			name: "partial score",
			update: func(scorer *peers.BlockProviderScorer) {
				// Partial score (less than a single batch of blocks processed).
				scorer.IncrementProcessedBlocks("peer1", batchSize/2)
			},
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "single batch",
			update: func(scorer *peers.BlockProviderScorer) {
				scorer.IncrementProcessedBlocks("peer1", batchSize)
			},
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, roundScore(0.05), scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "multiple batches",
			update: func(scorer *peers.BlockProviderScorer) {
				scorer.IncrementProcessedBlocks("peer1", batchSize*13)
			},
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, roundScore(0.05*13), scorer.Score("peer1"), "Unexpected score")
			},
		},
		{
			name: "maximum score cap",
			update: func(scorer *peers.BlockProviderScorer) {
				scorer.IncrementProcessedBlocks("peer1", batchSize*2)
				assert.Equal(t, roundScore(0.05*2), scorer.Score("peer1"), "Unexpected score")
				scorer.IncrementProcessedBlocks("peer1", scorer.Params().ProcessedBlocksCap)
			},
			check: func(scorer *peers.BlockProviderScorer) {
				assert.Equal(t, scorer.Params().ProcessedBlocksCap, scorer.ProcessedBlocks("peer1"))
				assert.Equal(t, 1.0, scorer.Score("peer1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				PeerLimit: 30,
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
						ProcessedBatchWeight: 0.05,
					},
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			if tt.update != nil {
				tt.update(scorer)
			}
			tt.check(scorer)
		})
	}
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
		name string
		cfg  *peers.BlockProviderScorerConfig
		want float64
	}{
		{
			name: "default config",
			cfg:  &peers.BlockProviderScorerConfig{},
			want: 1.0,
		},
		{
			name: "custom config",
			cfg: &peers.BlockProviderScorerConfig{
				ProcessedBatchWeight: 0.5,
				ProcessedBlocksCap:   batchSize * 300,
			},
			want: 150.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: tt.cfg,
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			assert.Equal(t, tt.want, scorer.MaxScore())
		})
	}
}

func TestPeerScorer_BlockProvider_FormatScorePretty(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	batchSize := uint64(flags.Get().BlockBatchLimit)
	format := "[%0.1f%%, raw: %0.2f,  blocks: %d/1280]"

	tests := []struct {
		name   string
		update func(s *peers.BlockProviderScorer)
		check  func(s *peers.BlockProviderScorer)
	}{
		{
			name:   "peer not registered",
			update: nil,
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 0.0, 0.0, 0), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "partial batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize/4)
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 0.0, 0.0, batchSize/4), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "single batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize)
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 5.0, 0.05, batchSize), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "3/2 of a batch",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*3/2)
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 5.0, 0.05, batchSize*3/2), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "multiple batches",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*5)
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 25.0, 0.05*5, batchSize*5), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "multiple batches max score",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", s.Params().ProcessedBlocksCap*5)
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 100.0, 1.0, s.Params().ProcessedBlocksCap), s.FormatScorePretty("peer1"))
			},
		},
		{
			name: "decaying",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementProcessedBlocks("peer1", batchSize*5)
				s.IncrementProcessedBlocks("peer1", batchSize)
				s.IncrementProcessedBlocks("peer1", batchSize/4)
				assert.Equal(t, fmt.Sprintf(format, 30.0, 0.05*6, batchSize*6+batchSize/4), s.FormatScorePretty("peer1"))
				// Maximize block count.
				s.IncrementProcessedBlocks("peer1", s.Params().ProcessedBlocksCap)
				assert.Equal(t, fmt.Sprintf(format, 100.0, 1.0, s.Params().ProcessedBlocksCap), s.FormatScorePretty("peer1"))
				// Half of blocks is to be decayed.
				s.Decay()
			},
			check: func(s *peers.BlockProviderScorer) {
				assert.Equal(t, fmt.Sprintf(format, 50.0, 0.5, s.Params().ProcessedBlocksCap/2), s.FormatScorePretty("peer1"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
						ProcessedBatchWeight: 0.05,
						ProcessedBlocksCap:   20 * batchSize,
					},
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			if tt.update != nil {
				tt.update(scorer)
			}
			tt.check(scorer)
		})
	}
}
