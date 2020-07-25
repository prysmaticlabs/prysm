package peers_test

import (
	"context"
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
		// Bad responses stats.
		assert.Equal(t, peers.DefaultBadResponsesThreshold, scorer.Params().BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, peers.DefaultBadResponsesWeight, scorer.Params().BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, peers.DefaultBadResponsesDecayInterval, scorer.Params().BadResponsesDecayInterval, "Unexpected decay interval value")
		// Block providers stats.
		assert.Equal(t, peers.DefaultBlockProviderReturnedBlocksWeight, scorer.Params().BlockProviderReturnedBlocksWeight)
		assert.Equal(t, peers.DefaultBlockProviderProcessedBlocksWeight, scorer.Params().BlockProviderProcessedBlocksWeight)
		assert.Equal(t, peers.DefaultBlockProviderDecayInterval, scorer.Params().BlockProviderDecayInterval)
		assert.Equal(t, peers.DefaultBlockProviderDecay, scorer.Params().BlockProviderDecay)
	})

	t.Run("explicit config", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold:              2,
				BadResponsesWeight:                 -1,
				BadResponsesDecayInterval:          1 * time.Minute,
				BlockProviderReturnedBlocksWeight:  0.5,
				BlockProviderProcessedBlocksWeight: 0.6,
				BlockProviderDecayInterval:         1 * time.Minute,
				BlockProviderDecay:                 0.8,
			},
		})
		scorer := peerStatuses.Scorer()
		// Bad responses stats.
		assert.Equal(t, 2, scorer.Params().BadResponsesThreshold, "Unexpected threshold value")
		assert.Equal(t, -1.0, scorer.Params().BadResponsesWeight, "Unexpected weight value")
		assert.Equal(t, 1*time.Minute, scorer.Params().BadResponsesDecayInterval, "Unexpected decay interval value")
		// Block providers stats.
		assert.Equal(t, 0.5, scorer.Params().BlockProviderReturnedBlocksWeight)
		assert.Equal(t, 0.6, scorer.Params().BlockProviderProcessedBlocksWeight)
		assert.Equal(t, 1*time.Minute, scorer.Params().BlockProviderDecayInterval)
		assert.Equal(t, 0.8, scorer.Params().BlockProviderDecay)
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

	setupScorer := func() (*peers.PeerScorer, []peer.ID) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			PeerLimit: 30,
			ScorerParams: &peers.PeerScorerConfig{
				BadResponsesThreshold:              5,
				BadResponsesWeight:                 -1.0,
				BlockProviderReturnedBlocksWeight:  0.1,
				BlockProviderProcessedBlocksWeight: 0.2,
				BlockProviderDecay:                 0.5, // 50% decay
			},
		})
		s := peerStatuses.Scorer()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			assert.Equal(t, float64(0), s.Score(pid), "Unexpected score")
		}
		assert.DeepEqual(t, map[string]float64{"peer1": 0, "peer2": 0, "peer3": 0}, peerScores(s, pids), "Unexpected scores")
		return s, pids
	}

	t.Run("bad responses score", func(t *testing.T) {
		s, pids := setupScorer()

		// Update peers' stats and test the effect on peer order.
		s.IncrementBadResponses("peer2")
		assert.DeepEqual(t, map[string]float64{"peer1": 0, "peer2": -0.2, "peer3": 0}, peerScores(s, pids), "Unexpected scores")
		s.IncrementBadResponses("peer1")
		s.IncrementBadResponses("peer1")
		assert.DeepEqual(t, map[string]float64{"peer1": -0.4, "peer2": -0.2, "peer3": 0}, peerScores(s, pids), "Unexpected scores")

		// See how decaying affects order of peers.
		s.DecayBadResponsesStats()
		assert.DeepEqual(t, map[string]float64{"peer1": -0.2, "peer2": 0, "peer3": 0}, peerScores(s, pids), "Unexpected scores")
		s.DecayBadResponsesStats()
		assert.DeepEqual(t, map[string]float64{"peer1": 0, "peer2": 0, "peer3": 0}, peerScores(s, pids), "Unexpected scores")
	})

	t.Run("block providers score", func(t *testing.T) {
		s, pids := setupScorer()

		// Make sure that until requested blocks is not 0, block provider score is unaffected.
		s.IncrementReturnedBlocks("peer1", 32)
		s.IncrementProcessedBlocks("peer1", 64)
		assert.Equal(t, float64(0), s.ScoreBlockProvider("peer1"), "Unexpected %q score", "peer1")

		// Now, with requested blocks counter updated, non-null score is expected.
		s.IncrementRequestedBlocks("peer1", 128)
		assert.DeepEqual(t, map[string]float64{"peer1": 0.125, "peer2": 0, "peer3": 0}, peerScores(s, pids), "Unexpected scores")

		// Test setting individual (returned blocks and processed blocks) scores.
		s.IncrementRequestedBlocks("peer2", 128)
		s.IncrementReturnedBlocks("peer2", 64)
		assert.DeepEqual(t, map[string]float64{"peer1": 0.125, "peer2": 0.05, "peer3": 0}, peerScores(s, pids), "Unexpected scores")
		s.IncrementRequestedBlocks("peer3", 128)
		s.IncrementProcessedBlocks("peer3", 64)
		assert.DeepEqual(t, map[string]float64{"peer1": 0.125, "peer2": 0.05, "peer3": 0.1}, peerScores(s, pids), "Unexpected scores")

		// See effect of decaying.
		assert.Equal(t, uint64(128), s.RequestedBlocks("peer1"), "Unexpected number of requested blocks in %q", "peer1")
		assert.Equal(t, uint64(128), s.RequestedBlocks("peer2"), "Unexpected number of requested blocks in %q", "peer2")
		assert.Equal(t, uint64(128), s.RequestedBlocks("peer3"), "Unexpected number of requested blocks in %q", "peer3")
		assert.Equal(t, uint64(32), s.ReturnedBlocks("peer1"), "Unexpected number of returned blocks in %q", "peer1")
		assert.Equal(t, uint64(64), s.ReturnedBlocks("peer2"), "Unexpected number of returned blocks in %q", "peer2")
		assert.Equal(t, uint64(0), s.ReturnedBlocks("peer3"), "Unexpected number of returned blocks in %q", "peer3")
		assert.Equal(t, uint64(64), s.ProcessedBlocks("peer1"), "Unexpected number of processed blocks in %q", "peer1")
		assert.Equal(t, uint64(0), s.ProcessedBlocks("peer2"), "Unexpected number of processed blocks in %q", "peer2")
		assert.Equal(t, uint64(64), s.ProcessedBlocks("peer3"), "Unexpected number of processed blocks in %q", "peer3")
		s.DecayBlockProvidersStats()
		assert.DeepEqual(t, map[string]float64{"peer1": 0.125, "peer2": 0.05, "peer3": 0.1}, peerScores(s, pids), "Unexpected scores")
		assert.Equal(t, uint64(128/2), s.RequestedBlocks("peer1"), "Unexpected number of requested blocks in %q", "peer1")
		assert.Equal(t, uint64(128/2), s.RequestedBlocks("peer2"), "Unexpected number of requested blocks in %q", "peer2")
		assert.Equal(t, uint64(128/2), s.RequestedBlocks("peer3"), "Unexpected number of requested blocks in %q", "peer3")
		assert.Equal(t, uint64(32/2), s.ReturnedBlocks("peer1"), "Unexpected number of returned blocks in %q", "peer1")
		assert.Equal(t, uint64(64/2), s.ReturnedBlocks("peer2"), "Unexpected number of returned blocks in %q", "peer2")
		assert.Equal(t, uint64(0/2), s.ReturnedBlocks("peer3"), "Unexpected number of returned blocks in %q", "peer3")
		assert.Equal(t, uint64(64/2), s.ProcessedBlocks("peer1"), "Unexpected number of processed blocks in %q", "peer1")
		assert.Equal(t, uint64(0/2), s.ProcessedBlocks("peer2"), "Unexpected number of processed blocks in %q", "peer2")
		assert.Equal(t, uint64(64/2), s.ProcessedBlocks("peer3"), "Unexpected number of processed blocks in %q", "peer3")
	})

	t.Run("overall score", func(t *testing.T) {
		// Full score, no penalty.
		s, _ := setupScorer()
		s.IncrementRequestedBlocks("peer1", 128)
		s.IncrementReturnedBlocks("peer1", 128)
		s.IncrementProcessedBlocks("peer1", 128)
		assert.Equal(t, 0.3, s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, float64(0), s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, 0.3, s.Score("peer1"), "Unexpected overall score")

		// Now, adjust score by introducing penalty for bad responses.
		s.IncrementBadResponses("peer1")
		assert.Equal(t, 0.3, s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, -0.2, s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, 0.1, s.Score("peer1"), "Unexpected overall score")
		// If peer continues to misbehave, score becomes negative.
		s.IncrementBadResponses("peer1")
		assert.Equal(t, 0.3, s.ScoreBlockProvider("peer1"), "Unexpected block provider score")
		assert.Equal(t, -0.4, s.ScoreBadResponses("peer1"), "Unexpected bad responses score")
		assert.Equal(t, -0.1, s.Score("peer1"), "Unexpected overall score")
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
