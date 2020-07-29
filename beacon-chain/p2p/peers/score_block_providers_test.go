package peers_test

import (
	"context"
	"math"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
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
				ReturnedBlocksWeight:       0.2,
				EmptyReturnedBatchPenalty:  -0.02,
				ProcessedBlocksWeight:      0.2,
				EmptyProcessedBatchPenalty: 0.0,
			},
		},
	})
	scorer := peerStatuses.Scorers().BlockProviderScorer()
	adjustScore := func(score float64) float64 {
		return math.Round((scorer.Params().StartScore+score)*10000) / 10000
	}

	assert.Equal(t, scorer.Params().StartScore, scorer.Score("peer1"), "Unexpected score for unregistered provider")
	// 128/64 = 2 batches of penalty is applied.
	scorer.IncrementRequestedBlocks("peer1", 128)
	assert.Equal(t, adjustScore(-0.04), scorer.Score("peer1"), "Unexpected score")
	// Now only processed blocks cause penalty (disabled as BlockProviderEmptyProcessedBatchPenalty: 0.0).
	scorer.IncrementReturnedBlocks("peer1", 64)
	assert.Equal(t, adjustScore(0.1), scorer.Score("peer1"), "Unexpected score")
	// Full score for returned blocks, penalty for processed blocks.
	scorer.IncrementReturnedBlocks("peer1", 64)
	assert.Equal(t, adjustScore(0.2), scorer.Score("peer1"), "Unexpected score")
	// No penalty, partial score.
	scorer.IncrementProcessedBlocks("peer1", 64)
	assert.Equal(t, adjustScore(0.2+0.1), scorer.Score("peer1"), "Unexpected score")
	// No penalty, full score.
	scorer.IncrementProcessedBlocks("peer1", 64)
	assert.Equal(t, adjustScore(0.2+0.2), scorer.Score("peer1"), "Unexpected score")
}

func TestPeerScorer_BlockProvider_GettersSetters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		ScorerParams: &peers.PeerScorerConfig{},
	})
	scorer := peerStatuses.Scorers().BlockProviderScorer()

	assert.Equal(t, uint64(0), scorer.RequestedBlocks("peer1"), "Unexpected count for unregistered peer")
	scorer.IncrementRequestedBlocks("peer1", 64)
	assert.Equal(t, uint64(64), scorer.RequestedBlocks("peer1"))

	assert.Equal(t, uint64(0), scorer.ReturnedBlocks("peer2"), "Unexpected count for unregistered peer")
	scorer.IncrementReturnedBlocks("peer2", 64)
	assert.Equal(t, uint64(64), scorer.ReturnedBlocks("peer2"))

	assert.Equal(t, uint64(0), scorer.ProcessedBlocks("peer3"), "Unexpected count for unregistered peer")
	scorer.IncrementProcessedBlocks("peer3", 64)
	assert.Equal(t, uint64(64), scorer.ProcessedBlocks("peer3"))
}

func TestPeerScorer_BlockProvider_Sorted(t *testing.T) {
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
				s.IncrementRequestedBlocks("peer1", 64)
				s.IncrementRequestedBlocks("peer2", 64)
				s.IncrementRequestedBlocks("peer3", 64)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer2", "peer3"},
		},
		{
			name: "different scores",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementRequestedBlocks("peer1", 64)
				s.IncrementRequestedBlocks("peer2", 128)
				s.IncrementRequestedBlocks("peer3", 64)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer3", "peer2"},
		},
		{
			name: "positive and negative scores",
			update: func(s *peers.BlockProviderScorer) {
				s.IncrementRequestedBlocks("peer1", 64)
				s.IncrementReturnedBlocks("peer1", 32)
				s.IncrementProcessedBlocks("peer1", 16)
				s.IncrementRequestedBlocks("peer2", 128)
				s.IncrementRequestedBlocks("peer3", 64)
				s.IncrementReturnedBlocks("peer3", 64)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer3", "peer1", "peer2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
				ScorerParams: &peers.PeerScorerConfig{
					BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
						EmptyReturnedBatchPenalty: -0.02,
					},
				},
			})
			scorer := peerStatuses.Scorers().BlockProviderScorer()
			tt.update(scorer)
			assert.DeepEqual(t, tt.want, scorer.Sorted(tt.have))
		})
	}
}
