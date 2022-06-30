package sync

import (
	"time"

	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/encoding/ssz/equality"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/trailofbits/go-mutexasserts"
)

const maxSidecarsPerSlot = maxBlocksPerSlot

// represents a possibly signed BlobsSidecar
type queuedBlobsSidecar struct {
	s         *ethpb.BlobsSidecar
	sig       []byte
	validated bool
}

func (s *queuedBlobsSidecar) IsSigned() bool {
	return s.sig != nil
}

func (s *queuedBlobsSidecar) AsSignedBlobsSidecar() *ethpb.SignedBlobsSidecar {
	return &ethpb.SignedBlobsSidecar{Message: s.s, Signature: s.sig}
}

// Delete sidecar from the list from the pending queue using the slot as key.
// Note: this helper is not thread safe.
func (s *Service) deleteSidecarFromPendingQueue(slot types.Slot, sc *queuedBlobsSidecar) error {
	mutexasserts.AssertRWMutexLocked(&s.pendingQueueLock)

	sidecars := s.pendingSidecarsInCache(slot)
	if len(sidecars) == 0 {
		return nil
	}

	newSidecars := make([]*queuedBlobsSidecar, 0, len(sidecars))
	for _, sidecar := range sidecars {
		if equality.DeepEqual(sidecar, sc) {
			continue
		}
		newSidecars = append(newSidecars, sidecar)
	}
	if len(newSidecars) == 0 {
		s.slotToPendingSidecars.Delete(slotToCacheKey(slot))
		delete(s.seenPendingSidecars, bytesutil.ToBytes32(sc.s.BeaconBlockRoot))
		return nil
	}

	// Decrease exp time in proportion to how many sidecars are still in the cache for slot key.
	d := pendingSidecarExpTime / time.Duration(len(newSidecars))
	if err := s.slotToPendingSidecars.Replace(slotToCacheKey(slot), newSidecars, d); err != nil {
		return err
	}
	delete(s.seenPendingSidecars, bytesutil.ToBytes32(sc.s.BeaconBlockRoot))
	return nil
}

// Insert sidecar to the list in the pending queue using its slot as key.
// Note: this helper is not thread safe.
func (s *Service) insertSidecarToPendingQueue(sidecar *queuedBlobsSidecar) {
	mutexasserts.AssertRWMutexLocked(&s.pendingQueueLock)

	root := bytesutil.ToBytes32(sidecar.s.BeaconBlockRoot)
	if s.seenPendingSidecars[root] {
		return
	}
	s.addPendingSidecarToCache(sidecar)
	s.seenPendingSidecars[root] = true
}

// This returns signed sidecars given input key from slotToPendingSidecars.
func (s *Service) pendingSidecarsInCache(slot types.Slot) []*queuedBlobsSidecar {
	k := slotToCacheKey(slot)
	value, ok := s.slotToPendingSidecars.Get(k)
	if !ok {
		return []*queuedBlobsSidecar{}
	}
	scs, ok := value.([]*queuedBlobsSidecar)
	if !ok {
		return []*queuedBlobsSidecar{}
	}
	return scs
}

// This adds input sidecar to slotToPendingSidecars cache.
func (s *Service) addPendingSidecarToCache(sc *queuedBlobsSidecar) {
	sidecars := s.pendingSidecarsInCache(sc.s.BeaconBlockSlot)
	if len(sidecars) >= maxSidecarsPerSlot {
		return
	}

	sidecars = append(sidecars, sc)
	k := slotToCacheKey(sc.s.BeaconBlockSlot)
	s.slotToPendingSidecars.Set(k, sidecars, pendingSidecarExpTime)
	return
}
