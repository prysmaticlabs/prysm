package peers_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
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
			assert.Equal(t, peers.DefaultBlockProviderProcessedBatchWeight, params.ProcessedBatchWeight)
			assert.Equal(t, peers.DefaultBlockProviderDecayInterval, params.DecayInterval)
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
					ProcessedBatchWeight: 0.6,
					DecayInterval:        1 * time.Minute,
					Decay:                16,
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
			assert.Equal(t, 0.6, params.ProcessedBatchWeight)
			assert.Equal(t, 1*time.Minute, params.DecayInterval)
			assert.Equal(t, uint64(16), params.Decay)
		})
	})
}

func TestPeerScorer_PeerScorerManager_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	batchSize := uint64(flags.Get().BlockBatchLimit)

	peerScores := func(s *peers.PeerScorerManager, pids []peer.ID) map[string]float64 {
		scores := make(map[string]float64, len(pids))
		for _, pid := range pids {
			scores[string(pid)] = s.Score(pid)
		}
		return scores
	}

	pack := func(scorer *peers.PeerScorerManager, s1, s2, s3 float64) map[string]float64 {
		return map[string]float64{
			"peer1": roundScore(s1),
			"peer2": roundScore(s2),
			"peer3": roundScore(s3),
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
					ProcessedBatchWeight: 0.05,
					Decay:                64,
				},
			},
		})
		s := peerStatuses.Scorers()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			// Not yet used peer gets boosted score.
			assert.Equal(t, 0.0, s.Score(pid), "Unexpected score for not yet used peer")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := peers.NewStatus(ctx, &peers.StatusConfig{
			ScorerParams: &peers.PeerScorerConfig{},
		})
		s := peerStatuses.Scorers()
		assert.Equal(t, 0.0, s.BadResponsesScorer().Score("peer1"))
		assert.Equal(t, 0.0, s.BlockProviderScorer().Score("peer1"))
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

		// Partial batch.
		s1.IncrementProcessedBlocks("peer1", batchSize/4)
		assert.Equal(t, 0.0, s.Score("peer1"), "Unexpected %q score", "peer1")

		// Single batch.
		s1.IncrementProcessedBlocks("peer1", batchSize)
		assert.DeepEqual(t, pack(s, 0.05, 0, 0), peerScores(s, pids), "Unexpected scores")

		// Multiple batches.
		s1.IncrementProcessedBlocks("peer2", batchSize*4)
		assert.DeepEqual(t, pack(s, 0.05, 0.05*4, 0), peerScores(s, pids), "Unexpected scores")

		// Partial batch.
		s1.IncrementProcessedBlocks("peer3", batchSize/2)
		assert.DeepEqual(t, pack(s, 0.05, 0.05*4, 0), peerScores(s, pids), "Unexpected scores")

		// See effect of decaying.
		assert.Equal(t, batchSize+batchSize/4, s1.ProcessedBlocks("peer1"))
		assert.Equal(t, batchSize*4, s1.ProcessedBlocks("peer2"))
		assert.Equal(t, batchSize/2, s1.ProcessedBlocks("peer3"))
		assert.DeepEqual(t, pack(s, 0.05, 0.05*4, 0), peerScores(s, pids), "Unexpected scores")
		s1.Decay()
		assert.Equal(t, batchSize/4, s1.ProcessedBlocks("peer1"))
		assert.Equal(t, batchSize*3, s1.ProcessedBlocks("peer2"))
		assert.Equal(t, uint64(0), s1.ProcessedBlocks("peer3"))
		assert.DeepEqual(t, pack(s, 0, 0.05*3, 0), peerScores(s, pids), "Unexpected scores")
	})

	t.Run("overall score", func(t *testing.T) {
		// Full score, no penalty.
		s, _ := setupScorer()
		s1 := s.BlockProviderScorer()
		s2 := s.BadResponsesScorer()

		s1.IncrementProcessedBlocks("peer1", batchSize*10)
		assert.Equal(t, roundScore(0.05*10), s1.Score("peer1"))
		// Now, adjust score by introducing penalty for bad responses.
		s2.Increment("peer1")
		s2.Increment("peer1")
		assert.Equal(t, -0.4, s2.Score("peer1"), "Unexpected bad responses score")
		assert.Equal(t, roundScore(0.05*10), s1.Score("peer1"), "Unexpected block provider score")
		assert.Equal(t, roundScore(0.05*10-0.4), s.Score("peer1"), "Unexpected overall score")
		// If peer continues to misbehave, score becomes negative.
		s2.Increment("peer1")
		assert.Equal(t, -0.6, s2.Score("peer1"), "Unexpected bad responses score")
		assert.Equal(t, roundScore(0.05*10), s1.Score("peer1"), "Unexpected block provider score")
		assert.Equal(t, -0.1, s.Score("peer1"), "Unexpected overall score")
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
				Decay:         64,
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

	s2.IncrementProcessedBlocks("peer1", 221)
	assert.Equal(t, uint64(221), s2.ProcessedBlocks("peer1"))

	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if s1.IsBadPeer(pid1) == false && s2.ProcessedBlocks("peer1") == 0 {
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
	assert.Equal(t, uint64(0), s2.ProcessedBlocks("peer1"), "No blocks are expected")
}
