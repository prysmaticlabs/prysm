package peers_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPeerScorer_ScoreBlockProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerConfig{
			BlockProviderReturnedBlocksWeight:     0.1,
			BlockProviderNoReturnedBlocksPenalty:  -0.1,
			BlockProviderProcessedBlocksWeight:    0.2,
			BlockProviderNoProcessedBlocksPenalty: -0.2,
		},
	})
	scorer := peerStatuses.Scorer()

	assert.Equal(t, 0.0, scorer.ScoreBlockProvider("peer1"), "Unexpected score for unregistered provider")
	scorer.IncrementRequestedBlocks("peer1", 128)
	assert.Equal(t, -0.3, scorer.ScoreBlockProvider("peer1"), "Unexpected score")
	scorer.IncrementReturnedBlocks("peer1", 64)
	assert.Equal(t, -0.15, scorer.ScoreBlockProvider("peer1"), "Unexpected score")
	scorer.IncrementReturnedBlocks("peer1", 64)
	assert.Equal(t, -0.1, scorer.ScoreBlockProvider("peer1"), "Unexpected score")
	scorer.IncrementProcessedBlocks("peer1", 64)
	assert.Equal(t, 0.2, scorer.ScoreBlockProvider("peer1"), "Unexpected score")
	scorer.IncrementProcessedBlocks("peer1", 64)
	assert.Equal(t, 0.3, scorer.ScoreBlockProvider("peer1"), "Unexpected score")
}

func TestPeerScorer_GettersSetters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		ScorerParams: &peers.PeerScorerConfig{},
	})
	scorer := peerStatuses.Scorer()

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

func TestPeerScorer_SortBlockProviders(t *testing.T) {
	tests := []struct {
		name   string
		change func(s *peers.PeerScorer)
		have   []peer.ID
		want   []peer.ID
	}{
		{
			name:   "no peers",
			change: func(s *peers.PeerScorer) {},
			have:   []peer.ID{},
			want:   []peer.ID{},
		},
		{
			name: "same scores",
			change: func(s *peers.PeerScorer) {
				s.IncrementRequestedBlocks("peer1", 64)
				s.IncrementRequestedBlocks("peer2", 64)
				s.IncrementRequestedBlocks("peer3", 64)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer2", "peer3"},
		},
		{
			name: "different scores",
			change: func(s *peers.PeerScorer) {
				s.IncrementRequestedBlocks("peer1", 64)
				s.IncrementRequestedBlocks("peer2", 128)
				s.IncrementRequestedBlocks("peer3", 64)
			},
			have: []peer.ID{"peer1", "peer2", "peer3"},
			want: []peer.ID{"peer1", "peer3", "peer2"},
		},
		{
			name: "positive and negative scores",
			change: func(s *peers.PeerScorer) {
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
				ScorerParams: &peers.PeerScorerConfig{},
			})
			scorer := peerStatuses.Scorer()
			tt.change(scorer)
			assert.DeepEqual(t, tt.want, scorer.SortBlockProviders(tt.have))
		})
	}
}
