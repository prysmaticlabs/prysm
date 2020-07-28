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
		// Block providers stats.
		assert.Equal(t, peers.DefaultBlockProviderReturnedBlocksWeight, params.BlockProviderReturnedBlocksWeight)
		assert.Equal(t, peers.DefaultBlockProviderEmptyReturnedBatchPenalty, params.BlockProviderEmptyReturnedBatchPenalty)
		assert.Equal(t, peers.DefaultBlockProviderProcessedBlocksWeight, params.BlockProviderProcessedBlocksWeight)
		assert.Equal(t, peers.DefaultBlockProviderEmptyProcessedBatchPenalty, params.BlockProviderEmptyProcessedBatchPenalty)
		assert.Equal(t, peers.DefaultBlockProviderDecayInterval, params.BlockProviderDecayInterval)
		assert.Equal(t, peers.DefaultBlockProviderDecay, params.BlockProviderDecay)
	})

	t.Run("explicit config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold:                   2,
				BadResponsesWeight:                      -1,
				BadResponsesDecayInterval:               1 * time.Minute,
				BlockProviderReturnedBlocksWeight:       0.5,
				BlockProviderEmptyReturnedBatchPenalty:  -0.2,
				BlockProviderProcessedBlocksWeight:      0.6,
				BlockProviderEmptyProcessedBatchPenalty: -0.3,
				BlockProviderDecayInterval:              1 * time.Minute,
				BlockProviderDecay:                      0.8,
			},
		})
		scorer := peerStatuses.Scorer()
		params := scorer.Params()
		// Bad responses stats.
		assert.Equal(t, 2, params.BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, -1.0, params.BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, 1*time.Minute, params.BadResponsesDecayInterval, "Unexpected decay interval value")
		// Block providers stats.
		assert.Equal(t, 0.5, params.BlockProviderReturnedBlocksWeight)
		assert.Equal(t, -0.2, params.BlockProviderEmptyReturnedBatchPenalty)
		assert.Equal(t, 0.6, params.BlockProviderProcessedBlocksWeight)
		assert.Equal(t, -0.3, params.BlockProviderEmptyProcessedBatchPenalty)
		assert.Equal(t, 1*time.Minute, params.BlockProviderDecayInterval)
		assert.Equal(t, 0.8, params.BlockProviderDecay)
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
		startScore := scorer.BlockProviderStartScore()
		maxScore := scorer.BlockProviderMaxScore()
		if scorer.RequestedBlocks("peer1") == 0 {
			s1 += maxScore
		} else {
			s1 += startScore
		}
		if scorer.RequestedBlocks("peer2") == 0 {
			s2 += maxScore
		} else {
			s2 += startScore
		}
		if scorer.RequestedBlocks("peer3") == 0 {
			s3 += maxScore
		} else {
			s3 += startScore
		}
		return map[string]float64{
			"peer1": math.Round(s1*10000) / 10000,
			"peer2": math.Round(s2*10000) / 10000,
			"peer3": math.Round(s3*10000) / 10000,
		}
	}

	adjustScore := func(scorer *peers.PeerScorer, score float64) float64 {
		startScore := scorer.BlockProviderStartScore()
		return math.Round((startScore+score)*10000) / 10000
	}

	setupScorer := func() (*peers.PeerScorer, []peer.ID) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold: 5,
				BlockProviderDecay:    0.5, // 50% decay
			},
		})
		s := peerStatuses.Scorer()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			assert.Equal(t, s.BlockProviderMaxScore(), s.Score(pid), "Unexpected score")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		})
		s := peerStatuses.Scorer()
		assert.Equal(t, 0.0, s.ScoreBadResponses("peer1"))
		assert.Equal(t, adjustScore(s, 0), s.ScoreBlockProvider("peer1"))
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

	t.Run("block providers score", func(t *testing.T) {
		s, pids := setupScorer()

		// Make sure that until requested blocks is not 0, block provider score is unaffected.
		s.IncrementReturnedBlocks("peer1", 32)
		s.IncrementProcessedBlocks("peer1", 64)
		assert.Equal(t, s.BlockProviderMaxScore(), s.ScoreBlockProvider("peer1"), "Unexpected %q score", "peer1")

		// Now, with requested blocks counter updated, non-null score is expected.
		s.IncrementRequestedBlocks("peer1", 128)
		assert.DeepEqual(t, pack(s, 0.12, 0, 0), peerScores(s, pids), "Unexpected scores")

		// Test setting individual (returned blocks and processed blocks) scores.
		s.IncrementRequestedBlocks("peer2", 128)
		s.IncrementReturnedBlocks("peer2", 64)
		assert.DeepEqual(t, pack(s, 0.12, 0.1, 0), peerScores(s, pids), "Unexpected scores")
		s.IncrementRequestedBlocks("peer3", 128)
		s.IncrementProcessedBlocks("peer3", 32)
		assert.DeepEqual(t, pack(s, 0.12, 0.1, 0.01), peerScores(s, pids), "Unexpected scores")

		// See effect of decaying.
		tests := map[peer.ID][2]struct {
			req, ret, proc uint64
		}{
			"peer1": {{128, 32, 64}, {128 / 2, 32 / 2, 64 / 2}},
			"peer2": {{128, 64, 0}, {128 / 2, 64 / 2, 0}},
			"peer3": {{128, 0, 32}, {128 / 2, 0 / 2, 32 / 2}},
		}
		for pid, counters := range tests {
			assert.Equal(t, counters[0].req, s.RequestedBlocks(pid), "Unexpected num. of requested blocks (%q)", string(pid))
			assert.Equal(t, counters[0].ret, s.ReturnedBlocks(pid), "Unexpected num. of returned blocks (%q)", string(pid))
			assert.Equal(t, counters[0].proc, s.ProcessedBlocks(pid), "Unexpected num. of processed blocks (%q)", string(pid))
		}
		assert.DeepEqual(t, pack(s, 0.12, 0.1, 0.01), peerScores(s, pids), "Unexpected scores")
		s.DecayBlockProvidersStats()
		// Stats decreased, penalty is not applied.
		assert.DeepEqual(t, pack(s, 0.15, 0.1, 0.05), peerScores(s, pids), "Unexpected scores")
		for pid, counters := range tests {
			assert.Equal(t, counters[1].req, s.RequestedBlocks(pid), "Unexpected num. of requested blocks (%q)", string(pid))
			assert.Equal(t, counters[1].ret, s.ReturnedBlocks(pid), "Unexpected num. of returned blocks (%q)", string(pid))
			assert.Equal(t, counters[1].proc, s.ProcessedBlocks(pid), "Unexpected num. of processed blocks (%q)", string(pid))
		}
	})

	t.Run("overall score", func(t *testing.T) {
		// Full score, no penalty.
		s, _ := setupScorer()
		s.IncrementRequestedBlocks("peer1", 128)
		s.IncrementReturnedBlocks("peer1", 128)
		s.IncrementProcessedBlocks("peer1", 128)
		assert.Equal(t, adjustScore(s, 0.4), s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, float64(0), s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, adjustScore(s, 0.4), s.Score("peer1"), "Unexpected overall score")

		// Now, adjust score by introducing penalty for bad responses.
		s.IncrementBadResponses("peer1")
		s.IncrementBadResponses("peer1")
		assert.Equal(t, -0.4, s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, adjustScore(s, 0.4), s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, adjustScore(s, 0.4-0.4), s.Score("peer1"), "Unexpected overall score")
		// If peer continues to misbehave, score becomes negative.
		s.IncrementBadResponses("peer1")
		assert.Equal(t, -0.6, s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, adjustScore(s, 0.4), s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, adjustScore(s, 0.4-0.6), s.Score("peer1"), "Unexpected overall score")
	})
}

func TestPeerScorer_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
		PeerLimit: 30,
		ScorerParams: &peers.PeerScorerConfig{
			BadResponsesThreshold:      5,
			BadResponsesWeight:         -0.5,
			BadResponsesDecayInterval:  50 * time.Millisecond,
			BlockProviderDecay:         0.95,
			BlockProviderDecayInterval: 25 * time.Millisecond,
		},
	})
	scorer := peerStatuses.Scorer()

	pid1 := peer.ID("peer1")
	peerStatuses.Add(nil, pid1, nil, network.DirUnknown)
	for i := 0; i < scorer.Params().BadResponsesThreshold+5; i++ {
		scorer.IncrementBadResponses(pid1)
	}
	assert.Equal(t, true, scorer.IsBadPeer(pid1), "Peer should be marked as bad")

	scorer.IncrementRequestedBlocks("peer1", 64)
	scorer.IncrementReturnedBlocks("peer1", 60)
	scorer.IncrementProcessedBlocks("peer1", 50)
	assert.NotEqual(t, 0.0, scorer.ScoreBlockProvider("peer1"))

	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if scorer.IsBadPeer(pid1) == false && scorer.ScoreBlockProvider("peer1") == 0 {
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
	assert.Equal(t, 0.0, scorer.ScoreBlockProvider("peer1"), "Peer should not have any block fetcher score")
}
