package peers_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
)

func TestPeerScorer_NewPeerScorer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("default params", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusParams{
			PeerLimit:    5,
			ScorerParams: &peers.PeerScorerParams{},
		})
		scorer := peerStatuses.Scorer()
		assert.Equal(t, peers.DefaultBadResponsesThreshold, scorer.Params().BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, peers.DefaultBadResponsesWeight, scorer.Params().BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, peers.DefaultBadResponsesDecayInterval, scorer.Params().BadResponsesDecayInterval, "Unexpected decay interval value")
	})

	t.Run("explicit params", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusParams{
			PeerLimit: 5,
			ScorerParams: &peers.PeerScorerParams{
				BadResponsesThreshold:     2,
				BadResponsesWeight:        -1,
				BadResponsesDecayInterval: 1 * time.Minute,
			},
		})
		scorer := peerStatuses.Scorer()
		assert.Equal(t, 2, scorer.Params().BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, -1.0, scorer.Params().BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, 1*time.Minute, scorer.Params().BadResponsesDecayInterval, "Unexpected decay interval value")
	})
}

func TestPeerScorer_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusParams{
		PeerLimit: 5,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold:     5,
			BadResponsesWeight:        -0.5,
			BadResponsesDecayInterval: 50 * time.Millisecond,
		},
	})
	scorer := peerStatuses.Scorer()

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
		peerStatuses.Add(nil, pid, nil, network.DirUnknown)
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
	scorer.DecayBadResponsesStats()
	assert.DeepEqual(t, []peer.ID{"peer2", "peer3", "peer1"}, sortByScore(pids), "Unexpected scores")
	scorer.DecayBadResponsesStats()
	assert.DeepEqual(t, []peer.ID{"peer1", "peer2", "peer3"}, sortByScore(pids), "Unexpected scores")
}

func TestPeerScorer_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusParams{
		PeerLimit: 5,
		ScorerParams: &peers.PeerScorerParams{
			BadResponsesThreshold:     5,
			BadResponsesWeight:        -0.5,
			BadResponsesDecayInterval: 50 * time.Millisecond,
		},
	})
	scorer := peerStatuses.Scorer()

	pid1 := peer.ID("peer1")
	peerStatuses.Add(nil, pid1, nil, network.DirUnknown)
	for i := 0; i < scorer.Params().BadResponsesThreshold+5; i++ {
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
