package peers

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestPeerScorer_BadResponsesThreshold(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxBadResponses := 2
	scorer := NewPeerScorer(ctx, &PeerScorerParams{
		BadResponsesThreshold: maxBadResponses,
	})
	assert.Equal(t, maxBadResponses, scorer.BadResponsesThreshold())
}

func TestPeerScorer_BadResponses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{})
	pid := peer.ID("peer1")
	_, err := scorer.BadResponses(pid)
	assert.ErrorContains(t, ErrPeerUnknown.Error(), err)

	scorer.AddPeer(pid)
	count, err := scorer.BadResponses(pid)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

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

func TestPeerScorer_IsBadPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{})
	pid := peer.ID("peer1")
	assert.Equal(t, false, scorer.IsBadPeer(pid))

	scorer.AddPeer(pid)
	assert.Equal(t, false, scorer.IsBadPeer(pid))

	for i := 0; i < defaultBadResponsesThreshold; i++ {
		scorer.IncrementBadResponses(pid)
		if i == defaultBadResponsesThreshold-1 {
			assert.Equal(t, true, scorer.IsBadPeer(pid), "Unexpected peer status")
		} else {
			assert.Equal(t, false, scorer.IsBadPeer(pid), "Unexpected peer status")
		}
	}
}

func TestPeerScorer_BadPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{})
	pids := []peer.ID{peer.ID("peer1"), peer.ID("peer2"), peer.ID("peer3"), peer.ID("peer4"), peer.ID("peer5")}
	for i := 0; i < len(pids); i++ {
		scorer.AddPeer(pids[i])
	}
	for i := 0; i < defaultBadResponsesThreshold; i++ {
		scorer.IncrementBadResponses(pids[1])
		scorer.IncrementBadResponses(pids[2])
		scorer.IncrementBadResponses(pids[4])
	}
	assert.Equal(t, false, scorer.IsBadPeer(pids[0]), "Invalid peer status")
	assert.Equal(t, true, scorer.IsBadPeer(pids[1]), "Invalid peer status")
	assert.Equal(t, true, scorer.IsBadPeer(pids[2]), "Invalid peer status")
	assert.Equal(t, false, scorer.IsBadPeer(pids[3]), "Invalid peer status")
	assert.Equal(t, true, scorer.IsBadPeer(pids[4]), "Invalid peer status")
	want := []peer.ID{pids[1], pids[2], pids[4]}
	badPeers := scorer.BadPeers()
	sort.Slice(badPeers, func(i, j int) bool {
		return badPeers[i] < badPeers[j]
	})
	assert.DeepEqual(t, want, badPeers, "Unexpected list of bad peers")
}
