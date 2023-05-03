package cache

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// RegistrationCache is used to store the cached results of an Validator Registration request.
// beacon api /eth/v1/validator/register_validator
type RegistrationCache struct {
	indexToRegistration map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1
	lock                sync.RWMutex
}

// NewRegistrationCache initializes the map and underlying cache.
func NewRegistrationCache() *RegistrationCache {
	return &RegistrationCache{
		indexToRegistration: make(map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1),
		lock:                sync.RWMutex{},
	}
}

// RegistrationByIndex returns the registration by index in the cache and also removes items in the cache if expired.
func (regCache *RegistrationCache) RegistrationByIndex(id primitives.ValidatorIndex) (*ethpb.ValidatorRegistrationV1, error) {
	regCache.lock.RLock()
	v, ok := regCache.indexToRegistration[id]
	if !ok {
		regCache.lock.RUnlock()
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	isExpired, err := RegistrationTimeStampExpired(v.Timestamp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check registration expiration")
	}
	if isExpired {
		regCache.lock.RUnlock()
		regCache.lock.Lock()
		defer regCache.lock.Unlock()
		delete(regCache.indexToRegistration, id)
		log.Warnf("registration for validator index %d expired at unix time %d", id, v.Timestamp)
		return nil, errors.Wrapf(ErrNotFoundRegistration, "validator id %d", id)
	}
	regCache.lock.RUnlock()
	return v, nil
}

func RegistrationTimeStampExpired(ts uint64) (bool, error) {
	// safely convert unint64 to int64
	i, err := math.Int(ts)
	if err != nil {
		return false, err
	}
	expiryDuration := params.BeaconConfig().RegistrationDuration
	// registered time + expiration duration < current time = expired
	return time.Unix(int64(i), 0).Add(expiryDuration).Before(time.Now()), nil
}

// UpdateIndexToRegisteredMap adds or updates values in the cache based on the argument.
func (regCache *RegistrationCache) UpdateIndexToRegisteredMap(ctx context.Context, m map[primitives.ValidatorIndex]*ethpb.ValidatorRegistrationV1) {
	_, span := trace.StartSpan(ctx, "RegistrationCache.UpdateIndexToRegisteredMap")
	defer span.End()
	regCache.lock.Lock()
	defer regCache.lock.Unlock()
	for key, value := range m {
		regCache.indexToRegistration[key] = &ethpb.ValidatorRegistrationV1{
			Pubkey:       bytesutil.SafeCopyBytes(value.Pubkey),
			FeeRecipient: bytesutil.SafeCopyBytes(value.FeeRecipient),
			GasLimit:     value.GasLimit,
			Timestamp:    value.Timestamp,
		}
	}
}
