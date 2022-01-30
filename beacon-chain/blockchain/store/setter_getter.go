package store

import ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

// PrevJustifiedCheckpt returns the previous justified checkpoint in the Store.
func (s *Store) PrevJustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.prevJustifiedCheckpt
}

// BestJustifiedCheckpt returns the best justified checkpoint in the Store.
func (s *Store) BestJustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.bestJustifiedCheckpt
}

// JustifiedCheckpt returns the justified checkpoint in the Store.
func (s *Store) JustifiedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.justifiedCheckpt
}

// PrevFinalizedCheckpt returns the previous finalized checkpoint in the Store.
func (s *Store) PrevFinalizedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.prevFinalizedCheckpt
}

// FinalizedCheckpt returns the finalized checkpoint in the Store.
func (s *Store) FinalizedCheckpt() *ethpb.Checkpoint {
	s.RLock()
	defer s.RUnlock()
	return s.finalizedCheckpt
}

// SetPrevJustifiedCheckpt sets the previous justified checkpoint in the Store.
func (s *Store) SetPrevJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.prevJustifiedCheckpt = cp
}

// SetBestJustifiedCheckpt sets the best justified checkpoint in the Store.
func (s *Store) SetBestJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.bestJustifiedCheckpt = cp
}

// SetJustifiedCheckpt sets the justified checkpoint in the Store.
func (s *Store) SetJustifiedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.justifiedCheckpt = cp
}

// SetFinalizedCheckpt sets the finalized checkpoint in the Store.
func (s *Store) SetFinalizedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.finalizedCheckpt = cp
}

// SetPrevFinalizedCheckpt sets the previous finalized checkpoint in the Store.
func (s *Store) SetPrevFinalizedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.prevFinalizedCheckpt = cp
}
