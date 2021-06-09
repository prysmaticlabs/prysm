package synccommittee

import (
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/copyutil"
)

// SaveSyncCommitteeSignature saves a sync committee signature in cache.
// The cache does not filter out duplicate signature, it will be up to the caller.
func (s *Store) SaveSyncCommitteeSignature(sig *ethpb.SyncCommitteeMessage) error {
	if sig == nil {
		return nilSignatureErr
	}

	copied := copyutil.CopySyncCommitteeSignature(sig)
	slot := copied.Slot
	s.signatureLock.Lock()
	defer s.signatureLock.Unlock()

	sigs, ok := s.signatureCache[slot]
	if !ok {
		s.signatureCache[slot] = []*ethpb.SyncCommitteeMessage{copied}
		return nil
	}

	s.signatureCache[slot] = append(sigs, copied)

	return nil
}

// SyncCommitteeSignatures returns sync committee signatures in cache by slot.
func (s *Store) SyncCommitteeSignatures(slot types.Slot) []*ethpb.SyncCommitteeMessage {
	s.signatureLock.RLock()
	defer s.signatureLock.RUnlock()

	sigs, ok := s.signatureCache[slot]
	if !ok {
		return nil
	}
	return sigs
}

// DeleteSyncCommitteeSignatures deletes sync committee signatures in cache by slot.
func (s *Store) DeleteSyncCommitteeSignatures(slot types.Slot) {
	s.signatureLock.Lock()
	defer s.signatureLock.Unlock()
	delete(s.signatureCache, slot)
}
