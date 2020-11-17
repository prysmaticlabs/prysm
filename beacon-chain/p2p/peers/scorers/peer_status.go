package scorers

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/peerdata"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
)

var _ Scorer = (*PeerStatusScorer)(nil)

// minSlotsTillHeadForUsefulPeer minimum number of processable slots for a peer to be considered useful.
const minSlotsTillHeadForUsefulPeer = 64 * 32

// PeerStatusScorer represents scorer that evaluates peers based on their statuses.
// Peer statuses are updated by regularly polling peers (see sync/rpc_status.go).
type PeerStatusScorer struct {
	config   *PeerStatusScorerConfig
	store    *peerdata.Store
	headSlot uint64
}

// PeerStatusScorerConfig holds configuration parameters for peer status scoring service.
type PeerStatusScorerConfig struct{}

// newPeerStatusScorer creates new peer status scoring service.
func newPeerStatusScorer(store *peerdata.Store, config *PeerStatusScorerConfig) *PeerStatusScorer {
	if config == nil {
		config = &PeerStatusScorerConfig{}
	}
	return &PeerStatusScorer{
		config: config,
		store:  store,
	}
}

// Score returns calculated peer score.
func (s *PeerStatusScorer) Score(pid peer.ID) float64 {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.score(pid)
}

// score is a lock-free version of Score.
func (s *PeerStatusScorer) score(pid peer.ID) float64 {
	if s.isBadPeer(pid) {
		return BadPeerScore
	}
	score := float64(0)
	if s.headSlot == 0 {
		return score
	}
	peerData, ok := s.store.PeerData(pid)
	if !ok || peerData.ChainState == nil {
		return score
	}
	if peerData.ChainState.HeadSlot < s.headSlot {
		return score
	}
	// Full score for peer that advertises far enough head.
	if (peerData.ChainState.HeadSlot - s.headSlot) >= minSlotsTillHeadForUsefulPeer {
		return 1.00
	}
	return score
}

// IsBadPeer states if the peer is to be considered bad.
func (s *PeerStatusScorer) IsBadPeer(pid peer.ID) bool {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.isBadPeer(pid)
}

// isBadPeer is lock-free version of IsBadPeer.
func (s *PeerStatusScorer) isBadPeer(pid peer.ID) bool {
	peerData, ok := s.store.PeerData(pid)
	if !ok {
		return false
	}
	// Mark peer as bad, if the latest error is one of the terminal ones.
	terminalErrs := []error{
		p2p.ErrWrongForkDigestVersion,
	}
	for _, err := range terminalErrs {
		if errors.Is(peerData.ChainStateValidationError, err) {
			return true
		}
	}
	return false
}

// BadPeers returns the peers that are considered bad.
func (s *PeerStatusScorer) BadPeers() []peer.ID {
	s.store.RLock()
	defer s.store.RUnlock()
	return []peer.ID{}
}

// SetPeerStatus sets chain state data for a given peer.
func (s *PeerStatusScorer) SetPeerStatus(pid peer.ID, chainState *pb.Status, validationError error) {
	s.store.Lock()
	defer s.store.Unlock()

	peerData := s.store.PeerDataGetOrCreate(pid)
	peerData.ChainState = chainState
	peerData.ChainStateLastUpdated = timeutils.Now()
	peerData.ChainStateValidationError = validationError
}

// PeerStatus gets the chain state of the given remote peer.
// This can return nil if there is no known chain state for the peer.
// This will error if the peer does not exist.
func (s *PeerStatusScorer) PeerStatus(pid peer.ID) (*pb.Status, error) {
	s.store.RLock()
	defer s.store.RUnlock()
	return s.peerStatus(pid)
}

// peerStatus lock-free version of PeerStatus.
func (s *PeerStatusScorer) peerStatus(pid peer.ID) (*pb.Status, error) {
	if peerData, ok := s.store.PeerData(pid); ok {
		return peerData.ChainState, nil
	}
	return nil, peerdata.ErrPeerUnknown
}

// SetHeadSlot updates known head slot.
func (s *PeerStatusScorer) SetHeadSlot(slot uint64) {
	s.headSlot = slot
}
