package store

import (
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

var (
	ErrNilCheckpoint = errors.New("nil checkpoint")
)

// PrevJustifiedCheckpt returns the previous justified checkpoint in the Store.
func (s *Store) PrevJustifiedCheckpt() (*ethpb.Checkpoint, error) {
	s.RLock()
	defer s.RUnlock()
	if s.prevJustifiedCheckpt == nil {
		return nil, ErrNilCheckpoint
	}
	return s.prevJustifiedCheckpt, nil
}

// BestJustifiedCheckpt returns the best justified checkpoint in the Store.
func (s *Store) BestJustifiedCheckpt() (*ethpb.Checkpoint, error) {
	s.RLock()
	defer s.RUnlock()
	if s.bestJustifiedCheckpt == nil {
		return nil, ErrNilCheckpoint
	}
	return s.bestJustifiedCheckpt, nil
}

// JustifiedCheckpt returns the justified checkpoint in the Store.
func (s *Store) JustifiedCheckpt() (*ethpb.Checkpoint, error) {
	s.RLock()
	defer s.RUnlock()
	if s.justifiedCheckpt == nil {
		return nil, ErrNilCheckpoint
	}
	return s.justifiedCheckpt, nil
}

// JustifiedPayloadBlockHash returns the justified payload block hash reflecting justified check point.
func (s *Store) JustifiedPayloadBlockHash() [32]byte {
	s.RLock()
	defer s.RUnlock()
	return s.justifiedPayloadBlockHash
}

// PrevFinalizedCheckpt returns the previous finalized checkpoint in the Store.
func (s *Store) PrevFinalizedCheckpt() (*ethpb.Checkpoint, error) {
	s.RLock()
	defer s.RUnlock()
	if s.prevFinalizedCheckpt == nil {
		return nil, ErrNilCheckpoint
	}
	return s.prevFinalizedCheckpt, nil
}

// FinalizedCheckpt returns the finalized checkpoint in the Store.
func (s *Store) FinalizedCheckpt() (*ethpb.Checkpoint, error) {
	s.RLock()
	defer s.RUnlock()
	if s.finalizedCheckpt == nil {
		return nil, ErrNilCheckpoint
	}
	return s.finalizedCheckpt, nil
}

// FinalizedPayloadBlockHash returns the finalized payload block hash reflecting finalized check point.
func (s *Store) FinalizedPayloadBlockHash() [32]byte {
	s.RLock()
	defer s.RUnlock()
	return s.finalizedPayloadBlockHash
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

// SetJustifiedCheckptAndPayloadHash sets the justified checkpoint and blockhash in the Store.
func (s *Store) SetJustifiedCheckptAndPayloadHash(cp *ethpb.Checkpoint, h [32]byte) {
	s.Lock()
	defer s.Unlock()
	s.justifiedCheckpt = cp
	s.justifiedPayloadBlockHash = h
}

// SetFinalizedCheckptAndPayloadHash sets the finalized checkpoint and blockhash in the Store.
func (s *Store) SetFinalizedCheckptAndPayloadHash(cp *ethpb.Checkpoint, h [32]byte) {
	s.Lock()
	defer s.Unlock()
	s.finalizedCheckpt = cp
	s.finalizedPayloadBlockHash = h
}

// SetPrevFinalizedCheckpt sets the previous finalized checkpoint in the Store.
func (s *Store) SetPrevFinalizedCheckpt(cp *ethpb.Checkpoint) {
	s.Lock()
	defer s.Unlock()
	s.prevFinalizedCheckpt = cp
}
