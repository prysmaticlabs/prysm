package peers

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPeerScorer_NewPeerScorer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{})
	assert.Equal(t, defaultBadResponsesThreshold, scorer.params.BadResponsesThreshold, "Unexpected threshold value")
	assert.Equal(t, defaultBadResponsesWeight, scorer.params.BadResponsesWeight, "Unexpected weight value")
	assert.Equal(t, defaultBadResponsesDecayInterval, scorer.params.BadResponsesDecayInterval, "Unexpected decay interval value")

	scorer = NewPeerScorer(ctx, &PeerScorerParams{
		BadResponsesThreshold:     2,
		BadResponsesWeight:        -1,
		BadResponsesDecayInterval: 1 * time.Minute,
	})
	assert.Equal(t, 2, scorer.params.BadResponsesThreshold, "Unexpected threshold value")
	assert.Equal(t, -1.0, scorer.params.BadResponsesWeight, "Unexpected weight value")
	assert.Equal(t, 1*time.Minute, scorer.params.BadResponsesDecayInterval, "Unexpected decay interval value")
}

func TestPeerScorer_AddPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{})
	assert.Equal(t, 0, len(scorer.peerStats), "Invalid number of peers")
	scorer.AddPeer("peer1")
	assert.Equal(t, 1, len(scorer.peerStats), "Invalid number of peers")
}

func TestPeerScorer_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{
		BadResponsesThreshold:     5,
		BadResponsesWeight:        -0.5,
		BadResponsesDecayInterval: 50 * time.Millisecond,
	})

	sortByScore := func(pids []peer.ID) []peer.ID {
		sort.Slice(pids, func(i, j int) bool {
			scr1, scr2 := scorer.Score(pids[i]), scorer.Score(pids[j])
			if scr1 == scr2 {
				// Sort by peer ID, whenever peers have equal score.
				return pids[i] < pids[j]
			}
			return scr1 > scr2
		})
		return pids
	}

	pids := []peer.ID{"peer1", "peer2", "peer3"}
	for _, pid := range pids {
		scorer.AddPeer(pid)
		if score := scorer.Score(pid); score < 0 {
			t.Errorf("Unexpected peer score, want: >=0, got: %v", score)
		}
	}
	assert.DeepEqual(t, []peer.ID{"peer1", "peer2", "peer3"}, sortByScore(pids), "Unexpected scores")

	// Update peers' stats and test the effect on peer order.
	scorer.IncrementBadResponses("peer2")
	assert.DeepEqual(t, []peer.ID{"peer1", "peer3", "peer2"}, sortByScore(pids), "Unexpected scores")
	scorer.IncrementBadResponses("peer1")
	scorer.IncrementBadResponses("peer1")
	assert.DeepEqual(t, []peer.ID{"peer3", "peer2", "peer1"}, sortByScore(pids), "Unexpected scores")

	// See how decaying affects order of peers.
	scorer.decayBadResponsesStats()
	assert.DeepEqual(t, []peer.ID{"peer2", "peer3", "peer1"}, sortByScore(pids), "Unexpected scores")
	scorer.decayBadResponsesStats()
	assert.DeepEqual(t, []peer.ID{"peer1", "peer2", "peer3"}, sortByScore(pids), "Unexpected scores")
}

func TestPeerScorer_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	scorer := NewPeerScorer(ctx, &PeerScorerParams{
		BadResponsesThreshold:     5,
		BadResponsesWeight:        -0.5,
		BadResponsesDecayInterval: 50 * time.Millisecond,
	})

	pid1 := peer.ID("peer1")
	scorer.AddPeer(pid1)
	for i := 0; i < scorer.params.BadResponsesThreshold+5; i++ {
		scorer.IncrementBadResponses(pid1)
	}
	assert.Equal(t, true, scorer.IsBadPeer(pid1), "Peer should be marked as bad")

	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if scorer.IsBadPeer(pid1) == false {
					return
				}
			case <-ctx.Done():
				t.Error("Timed out")
				return
			}
		}
	}()

	<-done
	assert.Equal(t, false, scorer.IsBadPeer(pid1), "Peer should not be marked as bad")
}
