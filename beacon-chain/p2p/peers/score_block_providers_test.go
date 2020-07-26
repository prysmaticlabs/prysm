package peers_test

import (
	"context"
	"testing"

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
