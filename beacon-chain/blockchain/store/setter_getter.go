package store

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// PrevJustifiedCheckpt returns the previous justified checkpoint in the store.
func (s *store) PrevJustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.prevJustifiedCheckpt
}

// BestJustifiedCheckpt returns the best justified checkpoint in the store.
func (s *store) BestJustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.bestJustifiedCheckpt
}

// JustifiedCheckpt returns the justified checkpoint in the store.
func (s *store) JustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.justifiedCheckpt
}

// FinalizedCheckpt returns the finalized checkpoint in the store.
func (s *store) FinalizedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.finalizedCheckpt
}

// SetPrevJustifiedCheckpt sets the previous justified checkpoint in the store.
func (s *store) SetPrevJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.prevJustifiedCheckpt = cp
}

// SetBestJustifiedCheckpt sets the best justified checkpoint in the store.
func (s *store) SetBestJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.bestJustifiedCheckpt = cp
}

// SetJustifiedCheckpt sets the justified checkpoint in the store.
func (s *store) SetJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.justifiedCheckpt = cp
}

// SetFinalizedCheckpt sets the finalized checkpoint in the store.
func (s *store) SetFinalizedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.finalizedCheckpt = cp
}
