package state_native

import (
	"bytes"
	"sync"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v5/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	consensus_types "github.com/prysmaticlabs/prysm/v5/consensus-types"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// finalizedValidatorIndexCache maintains a mapping from validator public keys to their indices within the beacon state.
// It includes a lastFinalizedIndex to track updates up to the last finalized validator index,
// and uses a mutex for concurrent read/write access to the cache.
type finalizedValidatorIndexCache struct {
	indexMap           map[[fieldparams.BLSPubkeyLength]byte]primitives.ValidatorIndex // Maps finalized BLS public keys to validator indices.
	lastFinalizedIndex int                                                             // Index of the last finalized validator to avoid reading validators before this point.
	sync.RWMutex
}

// newFinalizedValidatorIndexCache initializes a new validator index cache with an empty index map.
func newFinalizedValidatorIndexCache() *finalizedValidatorIndexCache {
	return &finalizedValidatorIndexCache{
		indexMap: make(map[[fieldparams.BLSPubkeyLength]byte]primitives.ValidatorIndex),
	}
}

// getValidatorIndex retrieves the validator index for a given public key from the cache.
// If the public key is not found in the cache, it searches through the state starting from the last finalized index.
func (b *BeaconState) getValidatorIndex(pubKey [fieldparams.BLSPubkeyLength]byte) (primitives.ValidatorIndex, bool) {
	b.validatorIndexCache.RLock()
	index, found := b.validatorIndexCache.indexMap[pubKey]
	b.validatorIndexCache.RUnlock()
	if found {
		return index, true
	}

	finalizedIndex := b.validatorIndexCache.lastFinalizedIndex
	vals, err := b.validatorsReadOnlySinceIndex(finalizedIndex)
	if err != nil {
		log.WithError(err).Errorf("Failed to get public keys since after validator index %d", finalizedIndex)
		return 0, false
	}
	for i, val := range vals {
		if bytes.Equal(val.PublicKeySlice(), pubKey[:]) {
			index := primitives.ValidatorIndex(finalizedIndex + i + 1)
			return index, true
		}
	}
	return 0, false
}

// saveValidatorIndices updates the validator index cache with new indices.
// It processes validator indices starting after the last finalized index and updates the tracker.
func (b *BeaconState) saveValidatorIndices() {
	b.validatorIndexCache.Lock()
	defer b.validatorIndexCache.Unlock()

	finalizedIndex := b.validatorIndexCache.lastFinalizedIndex
	vals, err := b.validatorsReadOnlySinceIndex(finalizedIndex)
	if err != nil {
		log.WithError(err).Errorf("Failed to retrieve public keys starting from the last finalized index %d", finalizedIndex)
		return
	}
	for i, val := range vals {
		b.validatorIndexCache.indexMap[val.PublicKey()] = primitives.ValidatorIndex(finalizedIndex + i + 1)
	}
	b.validatorIndexCache.lastFinalizedIndex += len(vals)
}

// validatorsReadOnlySinceIndex constructs a list of read only validator references after a specified index.
// The indices in the returned list correspond to their respective validator indices in the state.
// It returns an error if the specified index is out of bounds. This function is read-only and does not use locks.
func (b *BeaconState) validatorsReadOnlySinceIndex(index int) ([]state.ReadOnlyValidator, error) {
	totalValidators := b.validatorsLen()
	if index >= totalValidators {
		return nil, errors.Wrapf(consensus_types.ErrOutOfBounds, "index %d is out of bounds %d", index, totalValidators)
	}

	var v []*ethpb.Validator
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return nil, nil
		}
		v = b.validatorsMultiValue.Value(b)
	} else {
		if b.validators == nil {
			return nil, nil
		}
		v = b.validators
	}

	result := make([]state.ReadOnlyValidator, totalValidators-index)
	var err error
	for i := index + 1; i < totalValidators; i++ {
		val := v[i]
		if val == nil {
			continue
		}
		result[i], err = NewValidator(val)
		if err != nil {
			continue
		}
	}
	return result, nil
}
