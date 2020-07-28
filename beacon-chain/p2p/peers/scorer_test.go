package peers_test

import (
	"context"
	"math"
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

	t.Run("default config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit:    30,
			ScorerParams: &peers.PeerScorerConfig{},
		})
		scorer := peerStatuses.Scorer()
		params := scorer.Params()
		// Bad responses stats.
		assert.Equal(t, peers.DefaultBadResponsesThreshold, params.BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, peers.DefaultBadResponsesWeight, params.BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, peers.DefaultBadResponsesDecayInterval, params.BadResponsesDecayInterval, "Unexpected decay interval value")
	})

	t.Run("explicit config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold:     2,
				BadResponsesWeight:        -1,
				BadResponsesDecayInterval: 1 * time.Minute,
			},
		})
		scorer := peerStatuses.Scorer()
		params := scorer.Params()
		// Bad responses stats.
		assert.Equal(t, 2, params.BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, -1.0, params.BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, 1*time.Minute, params.BadResponsesDecayInterval, "Unexpected decay interval value")
	})
}

func TestPeerScorer_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerScores := func(s *peers.PeerScorer, pids []peer.ID) map[string]float64 {
		scores := make(map[string]float64, len(pids))
		for _, pid := range pids {
			scores[string(pid)] = s.Score(pid)
		}
		return scores
	}

	pack := func(scorer *peers.PeerScorer, s1, s2, s3 float64) map[string]float64 {
		return map[string]float64{
			"peer1": math.Round(s1*10000) / 10000,
			"peer2": math.Round(s2*10000) / 10000,
			"peer3": math.Round(s3*10000) / 10000,
		}
	}

	setupScorer := func() (*peers.PeerScorer, []peer.ID) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold: 5,
			},
		})
		s := peerStatuses.Scorer()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			assert.Equal(t, 0.0, s.Score(pid), "Unexpected score")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		})
		s := peerStatuses.Scorer()
		assert.Equal(t, 0.0, s.ScoreBadResponses("peer1"))
		assert.Equal(t, 0.0, s.Score("peer1"))
	})

	t.Run("bad responses score", func(t *testing.T) {
		s, pids := setupScorer()

		// Update peers' stats and test the effect on peer order.
		s.IncrementBadResponses("peer2")
		assert.DeepEqual(t, pack(s, 0, -0.2, 0), peerScores(s, pids), "Unexpected scores")
		s.IncrementBadResponses("peer1")
		s.IncrementBadResponses("peer1")
		assert.DeepEqual(t, pack(s, -0.4, -0.2, 0), peerScores(s, pids), "Unexpected scores")

		// See how decaying affects order of peers.
		s.DecayBadResponsesStats()
		assert.DeepEqual(t, pack(s, -0.2, 0, 0), peerScores(s, pids), "Unexpected scores")
		s.DecayBadResponsesStats()
		assert.DeepEqual(t, pack(s, 0, 0, 0), peerScores(s, pids), "Unexpected scores")
	})
}

func TestPeerScorer_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerConfig{
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
