package cache

import (
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

type syncSubnetIDs struct {
	sCommittee    *cache.Cache
	sCommiteeLock sync.RWMutex
}

// SyncSubnetIDs for sync committee participant.
var SyncSubnetIDs = newSyncSubnetIDs()

func newSyncSubnetIDs() *syncSubnetIDs {
	epochDuration := time.Duration(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot))
	subLength := epochDuration * time.Duration(params.BeaconConfig().EpochsPerSyncCommitteePeriod)
	persistentCache := cache.New(subLength*time.Second, epochDuration*time.Second)
	return &syncSubnetIDs{sCommittee: persistentCache}
}

// GetSyncCommitteeSubnets retrieves the sync committee subnet and expiration time of that validator's
// subscription.
func (s *syncSubnetIDs) GetSyncCommitteeSubnets(pubkey []byte) ([]uint64, uint64, bool, time.Time) {
	s.sCommiteeLock.RLock()
	defer s.sCommiteeLock.RUnlock()

	id, duration, ok := s.sCommittee.GetWithExpiration(string(pubkey))
	if !ok {
		return []uint64{}, 0, ok, time.Time{}
	}
	// Retrieve the slot from the cache.
	idxs, ok := id.([]uint64)
	if !ok {
		return []uint64{}, 0, ok, time.Time{}
	}
	// If no committees are saved, we return
	// nothing.
	if len(idxs) <= 1 {
		return []uint64{}, 0, ok, time.Time{}
	}
	return idxs[1:], idxs[0], ok, duration
}

// GetAllSubnets retrieves all the non-expired subscribed subnets of all the validators
// in the cache.
func (s *syncSubnetIDs) GetAllSubnets() []uint64 {
	s.sCommiteeLock.RLock()
	defer s.sCommiteeLock.RUnlock()

	itemsMap := s.sCommittee.Items()
	var committees []uint64

	for _, v := range itemsMap {
		if v.Expired() {
			continue
		}
		idxs, ok := v.Object.([]uint64)
		if !ok {
			continue
		}
		if len(idxs) <= 1 {
			continue
		}
		// Ignore the first index as that represents the
		// slot of the validator's assignments.
		committees = append(committees, idxs[1:]...)
	}
	return sliceutil.SetUint64(committees)
}

// AddSyncCommitteeSubnets adds the relevant committee for that particular validator along with its
// expiration period.
func (s *syncSubnetIDs) AddSyncCommitteeSubnets(pubkey []byte, slot uint64, comIndex []uint64, duration time.Duration) {
	s.sCommiteeLock.Lock()
	defer s.sCommiteeLock.Unlock()

	// Append the slot of the subnet into the first index here.
	s.sCommittee.Set(string(pubkey), append([]uint64{slot}, comIndex...), duration)
}

// EmptyAllCaches empties out all the related caches and flushes any stored
// entries on them. This should only ever be used for testing, in normal
// production, handling of the relevant subnets for each role is done
// separately.
func (s *syncSubnetIDs) EmptyAllCaches() {
	// Clear the cache.

	s.sCommiteeLock.Lock()
	s.sCommittee.Flush()
	s.sCommiteeLock.Unlock()
}
