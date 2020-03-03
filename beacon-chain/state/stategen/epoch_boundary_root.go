package stategen

// This sets an epoch boundary slot to root mapping.
// The slot is the key and the root is the value.
func (s *State) setEpochBoundaryRoot(slot uint64, root [32]byte) {
	s.epochBoundaryLock.Lock()
	defer s.epochBoundaryLock.Unlock()
	s.epochBoundarySlotToRoot[slot] = root
}

// This reads epoch boundary slot to root mapping.
func (s *State) epochBoundaryRoot(slot uint64) ([32]byte, bool) {
	s.epochBoundaryLock.RLock()
	defer s.epochBoundaryLock.RUnlock()
	r, ok := s.epochBoundarySlotToRoot[slot]
	return r, ok
}

// This deletes an entry of epoch boundary slot to root mapping.
func (s *State) deleteEpochBoundaryRoot(slot uint64) {
	s.epochBoundaryLock.Lock()
	defer s.epochBoundaryLock.Unlock()
	delete(s.epochBoundarySlotToRoot, slot)
}
