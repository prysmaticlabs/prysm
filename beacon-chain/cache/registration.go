package cache

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

// RegistrationCache is used to store the cached results of an Validator Registration request.
// beacon api /eth/v1/validator/register_validator
type RegistrationCache struct {
	sync.RWMutex
	indexToRegistration map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1
}

// NewRegistrationCache initializes the map and underlying cache.
func NewRegistrationCache() *RegistrationCache {
	return &RegistrationCache{
		indexToRegistration: make(map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1),
	}
}

// GetRegistrationByIndex returns the registration by index in the cache and also removes items in the cache if expired.
func (regCache *RegistrationCache) GetRegistrationByIndex(id primitives.ValidatorIndex) (*ethpb.ValidatorRegistrationV1, error) {
	regCache.RLock()
	defer regCache.RUnlock()
	v, ok := regCache.indexToRegistration[id]
	if !ok {
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	if timeStampExpired(v.Timestamp) {
		delete(regCache.indexToRegistration, id)
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	return v, nil
}

func timeStampExpired(ts uint64) bool {
	expiryDuration := time.Duration(params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*3) * time.Second
	if time.Unix(int64(ts), 0).Add(expiryDuration).Unix() < time.Now().Unix() {
		return true
	}
	return false
}

// UpdateIndexToRegisteredMap adds or updates values in the cache based on the argument.
func (regCache *RegistrationCache) UpdateIndexToRegisteredMap(m map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1) {
	regCache.RLock()
	defer regCache.RUnlock()
	for key, value := range m {
		regCache.indexToRegistration[key] = value
	}
}
