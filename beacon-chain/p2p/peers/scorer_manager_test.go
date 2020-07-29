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

	adjustScore := func(scorer *peers.PeerScorerManager, pid peer.ID, score float64) float64 {
		if scorer.BlockProviderScorer().RequestedBlocks(pid) == 0 {
			// No yet used peer gets score boost.
			score += scorer.BlockProviderScorer().MaxScore()
		} else {
			score += scorer.BlockProviderScorer().Params().StartScore
		}
		return math.Round(score*peers.ScoreRoundingFactor) / peers.ScoreRoundingFactor
	}

	pack := func(scorer *peers.PeerScorerManager, s1, s2, s3 float64) map[string]float64 {
		return map[string]float64{
			"peer1": adjustScore(scorer, "peer1", s1),
			"peer2": adjustScore(scorer, "peer2", s2),
			"peer3": adjustScore(scorer, "peer3", s3),
		}
	}

	setupScorer := func() (*peers.PeerScorerManager, []peer.ID) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesScorerConfig: &peers.BadResponsesScorerConfig{
					Threshold: 5,
				},
				BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
					StartScore:                 0.1,
					ReturnedBlocksWeight:       0.2,
					SlowReturnedBlocksPenalty:  -0.1,
					ProcessedBlocksWeight:      0.2,
					SlowProcessedBlocksPenalty: -0.1,
					Decay:                      0.5,
				},
			},
		})
		s := peerStatuses.Scorers()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			// Not yet used peer gets boosted score.
			assert.Equal(t, adjustScore(s, pid, 0.0), s.Score(pid), "Unexpected score for not yet used peer")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		})
		s := peerStatuses.Scorers()
		assert.Equal(t, 0.0, s.BadResponsesScorer().Score("peer1"))
		assert.Equal(t, adjustScore(s, "peer1", 0), s.BlockProviderScorer().Score("peer1"))
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

	t.Run("block providers score", func(t *testing.T) {
		s, pids := setupScorer()
		s1 := s.BlockProviderScorer()

		// Make sure that until requested blocks is not 0, block provider score is unaffected.
		s1.IncrementReturnedBlocks("peer1", 64)
		s1.IncrementProcessedBlocks("peer1", 32)
		assert.Equal(t, s1.MaxScore(), s.Score("peer1"), "Unexpected %q score", "peer1")

		// Now, with requested blocks counter updated, non-null score is expected.
		s1.IncrementRequestedBlocks("peer1", 128)
		assert.DeepEqual(t, pack(s, 0.05, 0, 0), peerScores(s, pids), "Unexpected scores")

		// Test setting individual (returned blocks and processed blocks) scores.
		s1.IncrementRequestedBlocks("peer2", 128)
		s1.IncrementReturnedBlocks("peer2", 64)
		assert.DeepEqual(t, pack(s, 0.05, -0.05, 0), peerScores(s, pids), "Unexpected scores")
		s1.IncrementRequestedBlocks("peer3", 128)
		s1.IncrementProcessedBlocks("peer3", 32)
		assert.DeepEqual(t, pack(s, 0.05, -0.05, -0.05), peerScores(s, pids), "Unexpected scores")

		// See effect of decaying.
		tests := map[peer.ID][2]struct {
			req, ret, proc uint64
		}{
			"peer1": {{128, 64, 32}, {128 / 2, 64 / 2, 32 / 2}},
			"peer2": {{128, 64, 0}, {128 / 2, 64 / 2, 0}},
			"peer3": {{128, 0, 32}, {128 / 2, 0 / 2, 32 / 2}},
		}
		for pid, counters := range tests {
			assert.Equal(t, counters[0].req, s1.RequestedBlocks(pid), "Unexpected num. of requested blocks (%q)", string(pid))
			assert.Equal(t, counters[0].ret, s1.ReturnedBlocks(pid), "Unexpected num. of returned blocks (%q)", string(pid))
			assert.Equal(t, counters[0].proc, s1.ProcessedBlocks(pid), "Unexpected num. of processed blocks (%q)", string(pid))
		}
		assert.DeepEqual(t, pack(s, 0.05, -0.05, -0.05), peerScores(s, pids), "Unexpected scores")
		s1.Decay()
		// Stats remains the same (while absolute numbers are decreased).
		assert.DeepEqual(t, pack(s, 0.05, -0.05, -0.05), peerScores(s, pids), "Unexpected scores")
		for pid, counters := range tests {
			assert.Equal(t, counters[1].req, s1.RequestedBlocks(pid), "Unexpected num. of requested blocks (%q)", string(pid))
			assert.Equal(t, counters[1].ret, s1.ReturnedBlocks(pid), "Unexpected num. of returned blocks (%q)", string(pid))
			assert.Equal(t, counters[1].proc, s1.ProcessedBlocks(pid), "Unexpected num. of processed blocks (%q)", string(pid))
		}
	})

	t.Run("overall score", func(t *testing.T) {
		// Full score, no penalty.
		s, _ := setupScorer()
		s1 := s.BlockProviderScorer()
		s2 := s.BadResponsesScorer()

		s1.IncrementRequestedBlocks("peer1", 128)
		s1.IncrementReturnedBlocks("peer1", 128)
		s1.IncrementProcessedBlocks("peer1", 128)
		assert.Equal(t, adjustScore(s, "peer1", 0.4), s1.Score("peer1"))
		// Now, adjust score by introducing penalty for bad responses.
		s2.Increment("peer1")
		s2.Increment("peer1")
		assert.Equal(t, -0.4, s2.Score("peer1"), "Unexpected bad responses score")
		assert.Equal(t, adjustScore(s, "peer1", 0.4), s1.Score("peer1"), "Unexpected block provider score")
		assert.Equal(t, adjustScore(s, "peer1", 0.4-0.4), s.Score("peer1"), "Unexpected overall score")
		// If peer continues to misbehave, score becomes negative.
		s2.Increment("peer1")
		assert.Equal(t, -0.6, s2.Score("peer1"), "Unexpected bad responses score")
		assert.Equal(t, adjustScore(s, "peer1", 0.4), s1.Score("peer1"), "Unexpected block provider score")
		assert.Equal(t, adjustScore(s, "peer1", 0.4-0.6), s.Score("peer1"), "Unexpected overall score")
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
			BlockProviderScorerConfig: &peers.BlockProviderScorerConfig{
				DecayInterval: 25 * time.Millisecond,
				Decay:         0.95,
			},
		},
	})
	s1 := peerStatuses.Scorers().BadResponsesScorer()
	s2 := peerStatuses.Scorers().BlockProviderScorer()

	pid1 := peer.ID("peer1")
	peerStatuses.Add(nil, pid1, nil, network.DirUnknown)
	for i := 0; i < s1.Params().Threshold+5; i++ {
		s1.Increment(pid1)
	}
	assert.Equal(t, true, s1.IsBadPeer(pid1), "Peer should be marked as bad")

	s2.IncrementRequestedBlocks("peer1", 64)
	s2.IncrementReturnedBlocks("peer1", 60)
	s2.IncrementProcessedBlocks("peer1", 50)
	assert.NotEqual(t, uint64(0), s2.RequestedBlocks("peer1"))
	assert.NotEqual(t, uint64(0), s2.ReturnedBlocks("peer1"))
	assert.NotEqual(t, uint64(0), s2.ProcessedBlocks("peer1"))

	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if s1.IsBadPeer(pid1) == false && s2.RequestedBlocks("peer1") == 0 {
					return
				}
			case <-ctx.Done():
				t.Error("Timed out")
				return
			}
		}
	}()

	<-done
	assert.Equal(t, false, s1.IsBadPeer(pid1), "Peer should not be marked as bad")
	assert.Equal(t, uint64(0), s2.RequestedBlocks("peer1"), "No blocks are expected")
	assert.Equal(t, uint64(0), s2.ReturnedBlocks("peer1"), "No blocks are expected")
	assert.Equal(t, uint64(0), s2.ProcessedBlocks("peer1"), "No blocks are expected")
}
