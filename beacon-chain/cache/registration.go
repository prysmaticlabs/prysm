package cache

import (
	"math/big"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
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
	v, ok := regCache.indexToRegistration[id]
	if !ok {
		defer regCache.RUnlock()
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	if RegistrationTimeStampExpired(v.Timestamp) {
		regCache.RUnlock()
		regCache.Lock()
		defer regCache.Unlock()
		delete(regCache.indexToRegistration, id)
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	defer regCache.RUnlock()
	return v, nil
}

func RegistrationTimeStampExpired(ts uint64) bool {
	expiryDuration := time.Duration(params.BeaconConfig().SecondsPerSlot*uint64(params.BeaconConfig().SlotsPerEpoch)*3) * time.Second
	// safely convert unint64 to int64
	t := new(big.Int).SetUint64(ts).Int64()
	return time.Unix(t, 0).Add(expiryDuration).Unix() < time.Now().Unix()
}

// UpdateIndexToRegisteredMap adds or updates values in the cache based on the argument.
func (regCache *RegistrationCache) UpdateIndexToRegisteredMap(m map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1) {
	regCache.Lock()
	defer regCache.Unlock()
	for key, value := range m {
		regCache.indexToRegistration[key] = &ethpb.ValidatorRegistrationV1{
			Pubkey:       bytesutil.SafeCopyBytes(value.Pubkey),
			FeeRecipient: bytesutil.SafeCopyBytes(value.FeeRecipient),
			GasLimit:     value.GasLimit,
			Timestamp:    value.Timestamp,
		}
	}
}
