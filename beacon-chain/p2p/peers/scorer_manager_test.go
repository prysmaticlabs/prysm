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

func TestPeerScorer_PeerScorerManager_Init(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("default config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit:    30,
			ScorerParams: &peers.PeerScorerConfig{},
		})

		t.Run("bad responses scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BadResponsesScorer().Params()
			assert.Equal(t, peers.DefaultBadResponsesThreshold, params.Threshold, "Unexpected threshold value")
			assert.Equal(t, peers.DefaultBadResponsesWeight, params.Weight, "Unexpected weight value")
			assert.Equal(t, peers.DefaultBadResponsesDecayInterval, params.DecayInterval, "Unexpected decay interval value")
		})

		t.Run("block providers scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BlockProviderScorer().Params()
			assert.Equal(t, peers.DefaultBlockProviderReturnedBlocksWeight, params.ReturnedBlocksWeight)
			assert.Equal(t, peers.DefaultSlowReturnedBlocksPenalty, params.SlowReturnedBlocksPenalty)
			assert.Equal(t, peers.DefaultBlockProviderProcessedBlocksWeight, params.ProcessedBlocksWeight)
			assert.Equal(t, peers.DefaultSlowProcessedBlocksPenalty, params.SlowProcessedBlocksPenalty)
			assert.Equal(t, peers.DefaultBlockProviderDecayInterval, params.DecayInterval)
			assert.Equal(t, peers.DefaultBlockProviderDecay, params.Decay)
		})
	})

	t.Run("explicit config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesScorerConfig: &peers.BadResponsesScorerConfig{
					Threshold:     2,
					Weight:        -1,
					DecayInterval: 1 * time.Minute,
				},
				BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
					StartScore:                 0.2,
					ReturnedBlocksWeight:       0.5,
					SlowReturnedBlocksPenalty:  -0.2,
					ProcessedBlocksWeight:      0.6,
					SlowProcessedBlocksPenalty: -0.3,
					DecayInterval:              1 * time.Minute,
					Decay:                      0.8,
				},
			},
		})

		t.Run("bad responses scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BadResponsesScorer().Params()
			assert.Equal(t, 2, params.Threshold, "Unexpected threshold value")
			assert.Equal(t, -1.0, params.Weight, "Unexpected weight value")
			assert.Equal(t, 1*time.Minute, params.DecayInterval, "Unexpected decay interval value")
		})

		t.Run("block provider scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BlockProviderScorer().Params()
			assert.Equal(t, 0.2, params.StartScore)
			assert.Equal(t, 0.5, params.ReturnedBlocksWeight)
			assert.Equal(t, -0.2, params.SlowReturnedBlocksPenalty)
			assert.Equal(t, 0.6, params.ProcessedBlocksWeight)
			assert.Equal(t, -0.3, params.SlowProcessedBlocksPenalty)
			assert.Equal(t, 1*time.Minute, params.DecayInterval)
			assert.Equal(t, 0.8, params.Decay)
		})
	})
}

func TestPeerScorer_PeerScorerManager_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerScores := func(s *peers.PeerScorerManager, pids []peer.ID) map[string]float64 {
		scores := make(map[string]float64, len(pids))
		for _, pid := range pids {
			scores[string(pid)] = s.Score(pid)
		}
		return scores
	}

	pack := func(scorer *peers.PeerScorerManager, s1, s2, s3 float64) map[string]float64 {
		return map[string]float64{
			"peer1": math.Round(s1*10000) / 10000,
			"peer2": math.Round(s2*10000) / 10000,
			"peer3": math.Round(s3*10000) / 10000,
		}
	}

	setupScorer := func() (*peers.PeerScorerManager, []peer.ID) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesScorerConfig: &peers.BadResponsesScorerConfig{
					Threshold: 5,
				},
			},
		})
		s := peerStatuses.Scorers()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			// Not yet used peer gets boosted score.
			assert.Equal(t, s.BlockProviderScorer().MaxScore(), s.Score(pid), "Unexpected score for not yet used peer")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		})
		s := peerStatuses.Scorers()
		assert.Equal(t, 0.0, s.BadResponsesScorer().Score("peer1"))
		assert.Equal(t, 0.0, s.Score("peer1"))
	})

	t.Run("bad responses score", func(t *testing.T) {
		s, pids := setupScorer()

		// Update peers' stats and test the effect on peer order.
		s.BadResponsesScorer().Increment("peer2")
		assert.DeepEqual(t, pack(s, 0, -0.2, 0), peerScores(s, pids), "Unexpected scores")
		s.BadResponsesScorer().Increment("peer1")
		s.BadResponsesScorer().Increment("peer1")
		assert.DeepEqual(t, pack(s, -0.4, -0.2, 0), peerScores(s, pids), "Unexpected scores")

		// See how decaying affects order of peers.
		s.BadResponsesScorer().Decay()
		assert.DeepEqual(t, pack(s, -0.2, 0, 0), peerScores(s, pids), "Unexpected scores")
		s.BadResponsesScorer().Decay()
		assert.DeepEqual(t, pack(s, 0, 0, 0), peerScores(s, pids), "Unexpected scores")
	})
}

func TestPeerScorer_PeerScorerManager_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerConfig{
			BadResponsesScorerConfig: &peers.BadResponsesScorerConfig{
				Threshold:     5,
				Weight:        -0.5,
				DecayInterval: 50 * time.Millisecond,
			},
		},
	})
	scorer := peerStatuses.Scorers().BadResponsesScorer()

	pid1 := peer.ID("peer1")
	peerStatuses.Add(nil, pid1, nil, network.DirUnknown)
	for i := 0; i < scorer.Params().Threshold+5; i++ {
		scorer.Increment(pid1)
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
