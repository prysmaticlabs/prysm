package scorers

import (
	"context"
	"math"
	"reflect"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/peerdata"
)

// ScoreRoundingFactor defines how many digits to keep in decimal part.
// This parameter is used in math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor.
const ScoreRoundingFactor = 10000

const (
	scorerBadResponses  scorerID = "*scorers.BadResponsesScorer"
	scorerBlockProvider scorerID = "*scorers.BlockProviderScorer"
	scorerPeerStatus    scorerID = "*scorers.PeerStatusScorer"
)

// scorerID is used to distinguish between different scorers.
type scorerID string

// Scorer defines minimum set of methods every peer scorer must expose.
type Scorer interface {
	Score(pid peer.ID) float64
	IsBadPeer(pid peer.ID) bool
	BadPeers() []peer.ID
	Decay()
}

// Service manages peer scorers that are used to calculate overall peer score.
type Service struct {
	ctx         context.Context
	store       *peerdata.Store
	scorers     map[scorerID]Scorer
	weights     map[scorerID]float64
	totalWeight float64
}

// Config holds configuration parameters for scoring service.
type Config struct {
	BadResponsesScorerConfig  *BadResponsesScorerConfig
	BlockProviderScorerConfig *BlockProviderScorerConfig
	PeerStatusScorerConfig    *PeerStatusScorerConfig
}

// NewService provides fully initialized peer scoring service.
func NewService(ctx context.Context, store *peerdata.Store, config *Config) *Service {
	s := &Service{
		ctx:     ctx,
		store:   store,
		scorers: make(map[scorerID]Scorer),
		weights: make(map[scorerID]float64),
	}
	// Register scorers.
	s.registerScorer(newBadResponsesScorer(ctx, store, config.BadResponsesScorerConfig), 1.0)
	s.registerScorer(newBlockProviderScorer(ctx, store, config.BlockProviderScorerConfig), 1.0)
	s.registerScorer(newPeerStatusScorer(ctx, store, config.PeerStatusScorerConfig), 0.0)

	// Start background tasks.
	go s.loop(s.ctx)

	return s
}

// BadResponsesScorer exposes bad responses scoring service.
func (s *Service) BadResponsesScorer() *BadResponsesScorer {
	return s.scorers[scorerBadResponses].(*BadResponsesScorer)
}

// BlockProviderScorer exposes block provider scoring service.
func (s *Service) BlockProviderScorer() *BlockProviderScorer {
	return s.scorers[scorerBlockProvider].(*BlockProviderScorer)
}

// PeerStatusScorer exposes peer chain status scoring service.
func (s *Service) PeerStatusScorer() *PeerStatusScorer {
	return s.scorers[scorerPeerStatus].(*PeerStatusScorer)
}

// Count returns number of scorers that can affect score (have non-zero weight).
func (s *Service) ActiveScorersCount() int {
	cnt := 0
	for _, w := range s.weights {
		if w > 0 {
			cnt++
		}
	}
	return cnt
}

// Score returns calculated peer score across all tracked metrics.
func (s *Service) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()

	score := float64(0)
	if _, ok := s.store.PeerData(pid); !ok {
		return 0
	}
	score += s.BadResponsesScorer().score(pid) * s.scorerWeightFactor(scorerBadResponses)
	score += s.BlockProviderScorer().score(pid) * s.scorerWeightFactor(scorerBlockProvider)
	return math.Round(score*ScoreRoundingFactor) / ScoreRoundingFactor
}

// loop handles background tasks.
func (s *Service) loop(ctx context.Context) {
	decayBadResponsesStats := time.NewTicker(s.BadResponsesScorer().Params().DecayInterval)
	defer decayBadResponsesStats.Stop()
	decayBlockProviderStats := time.NewTicker(s.BlockProviderScorer().Params().DecayInterval)
	defer decayBlockProviderStats.Stop()

	for {
		select {
		case <-decayBadResponsesStats.C:
			s.BadResponsesScorer().Decay()
		case <-decayBlockProviderStats.C:
			s.BlockProviderScorer().Decay()
		case <-ctx.Done():
			return
		}
	}
}

// registerScorer adds scorer to map of known scorers.
func (s *Service) registerScorer(scorer Scorer, weight float64) {
	key := scorerID(reflect.TypeOf(scorer).String())
	s.scorers[key] = scorer
	s.weights[key] = weight
	s.totalWeight += s.weights[key]
}

// scorerWeightFactor calculates contribution percentage of a given scorer in total score.
func (s *Service) scorerWeightFactor(id scorerID) float64 {
	return s.weights[id] / s.totalWeight
}
