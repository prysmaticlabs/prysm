package cache

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db/kv"
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

// GetRegistrationByIndex
func (regCache *RegistrationCache) GetRegistrationByIndex(id primitives.ValidatorIndex) (*ethpb.ValidatorRegistrationV1, error) {
	regCache.RLock()
	defer regCache.RUnlock()
	v, ok := regCache.indexToRegistration[id]
	if !ok {
		return nil, errors.Wrapf(kv.ErrNotFoundFeeRecipient, "validator id %d", id)
	}
	if timeStampExpired(v.Timestamp) {
		delete(regCache.indexToRegistration, id)
		return nil, errors.Wrapf(kv.ErrNotFoundFeeRecipient, "validator id %d", id)
	}
	return v, nil
}

func timeStampExpired(ts uint64) bool {
	expiryDuration := time.Duration(params.BeaconConfig().SecondsPerSlot * uint64(params.BeaconConfig().SlotsPerEpoch) * 3)
	if time.Unix(int64(ts), 0).Add(expiryDuration).Unix() < time.Now().Unix() {
		return true
	}
	return false
}

// UpdateIndexToRegisteredMap
func (regCache *RegistrationCache) UpdateIndexToRegisteredMap(m map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1) {
	regCache.RLock()
	defer regCache.RUnlock()
	for key, value := range m {
		regCache.indexToRegistration[key] = value
	}
}
