package peers

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPeerScorer_decayBadResponsesStats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxBadResponses := 2
	scorer := NewPeerScorer(ctx, &PeerScorerParams{
		BadResponsesThreshold:     maxBadResponses,
		BadResponsesWeight:        1,
		BadResponsesDecayInterval: 50 * time.Nanosecond,
	})

	// Peer 1 has 0 bad responses.
	pid1 := peer.ID("peer1")
	scorer.AddPeer(pid1)
	badResponsesCount, err := scorer.BadResponses(pid1)
	require.NoError(t, err)
	assert.Equal(t, 0, badResponsesCount)

	// Peer 2 has 1 bad response.
	pid2 := peer.ID("peer2")
	scorer.AddPeer(pid2)
	scorer.IncrementBadResponses(pid2)
	badResponsesCount, err = scorer.BadResponses(pid2)
	require.NoError(t, err)
	assert.Equal(t, 1, badResponsesCount)

	// Peer 3 has 2 bad response.
	pid3 := peer.ID("peer3")
	scorer.AddPeer(pid3)
	scorer.IncrementBadResponses(pid3)
	scorer.IncrementBadResponses(pid3)
	badResponsesCount, err = scorer.BadResponses(pid3)
	require.NoError(t, err)
	assert.Equal(t, 2, badResponsesCount)

	// Decay the values
	scorer.decayBadResponsesStats()

	// Ensure the new values are as expected
	badResponsesCount, err = scorer.BadResponses(pid1)
	require.NoError(t, err)
	assert.Equal(t, 0, badResponsesCount, "unexpected bad responses for pid1")

	badResponsesCount, err = scorer.BadResponses(pid2)
	require.NoError(t, err)
	assert.Equal(t, 0, badResponsesCount, "unexpected bad responses for pid2")

	badResponsesCount, err = scorer.BadResponses(pid3)
	require.NoError(t, err)
	assert.Equal(t, 1, badResponsesCount, "unexpected bad responses for pid3")
}
