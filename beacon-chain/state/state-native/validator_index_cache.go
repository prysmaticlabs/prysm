package state_native

import (
	"bytes"
	"sync"

	"github.com/prysmaticlabs/prysm/v5/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
)

// finalizedValidatorIndexCache maintains a mapping from validator public keys to their indices within the beacon state.
// It includes a lastFinalizedIndex to track updates up to the last finalized validator index,
// and uses a mutex for concurrent read/write access to the cache.
type finalizedValidatorIndexCache struct {
	indexMap map[[fieldparams.BLSPubkeyLength]byte]primitives.ValidatorIndex // Maps finalized BLS public keys to validator indices.
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

	validatorCount := len(b.validatorIndexCache.indexMap)
	vals := b.validatorsReadOnlySinceIndex(validatorCount)
	for i, val := range vals {
		if bytes.Equal(bytesutil.PadTo(val.publicKeySlice(), 48), pubKey[:]) {
			index := primitives.ValidatorIndex(validatorCount + i)
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

	validatorCount := len(b.validatorIndexCache.indexMap)
	vals := b.validatorsReadOnlySinceIndex(validatorCount)
	for i, val := range vals {
		b.validatorIndexCache.indexMap[val.PublicKey()] = primitives.ValidatorIndex(validatorCount + i)
	}
}

// validatorsReadOnlySinceIndex constructs a list of read only validator references after a specified index.
// The indices in the returned list correspond to their respective validator indices in the state.
// It returns nil if the specified index is out of bounds. This function is read-only and does not use locks.
func (b *BeaconState) validatorsReadOnlySinceIndex(index int) []readOnlyValidator {
	totalValidators := b.validatorsLen()
	if index >= totalValidators {
		return nil
	}

	var v []*ethpb.Validator
	if features.Get().EnableExperimentalState {
		if b.validatorsMultiValue == nil {
			return nil
		}
		v = b.validatorsMultiValue.Value(b)
	} else {
		if b.validators == nil {
			return nil
		}
		v = b.validators
	}

	result := make([]readOnlyValidator, totalValidators-index)
	for i := 0; i < len(result); i++ {
		val := v[i+index]
		if val == nil {
			continue
		}
		result[i] = readOnlyValidator{
			validator: val,
		}
	}
	return result
}
