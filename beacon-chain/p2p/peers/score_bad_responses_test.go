package peers

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
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
	// Peer 2 has 1 bad response.
	pid2 := peer.ID("peer2")
	scorer.AddPeer(pid2)
	scorer.IncrementBadResponses(pid2)
	// Peer 3 has 2 bad response.
	pid3 := peer.ID("peer3")
	scorer.AddPeer(pid3)
	scorer.IncrementBadResponses(pid3)
	scorer.IncrementBadResponses(pid3)

	// Decay the values
	scorer.decayBadResponsesStats()

	// Ensure the new values are as expected
	badResponses1, err := scorer.BadResponses(pid1)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses1 != 0 {
		t.Errorf("Unexpected bad responses for peer 1: expected 0, received %v", badResponses1)
	}
	badResponses2, err := scorer.BadResponses(pid2)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses2 != 0 {
		t.Errorf("Unexpected bad responses for peer 2: expected 0, received %v", badResponses2)
	}
	badResponses3, err := scorer.BadResponses(pid3)
	if err != nil {
		t.Fatal(err)
	}
	if badResponses3 != 1 {
		t.Errorf("Unexpected bad responses for peer 3: expected 1, received %v", badResponses3)
	}
}
