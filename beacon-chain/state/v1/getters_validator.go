package v1

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/shared/copyutil"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorIndexOutOfRangeError represents an error scenario where a validator does not exist
// at a given index in the validator's array.
type ValidatorIndexOutOfRangeError struct {
	message string
}

var (
	// ErrNilValidatorsInState returns when accessing validators in the state while the state has a
	// nil slice for the validators field.
	ErrNilValidatorsInState = errors.New("state has nil validator slice")
)

// NewValidatorIndexOutOfRangeError creates a new error instance.
func NewValidatorIndexOutOfRangeError(index types.ValidatorIndex) ValidatorIndexOutOfRangeError {
	return ValidatorIndexOutOfRangeError{
		message: fmt.Sprintf("index %d out of range", index),
	}
}

// Error returns the underlying error message.
func (e *ValidatorIndexOutOfRangeError) Error() string {
	return e.message
}

// Validators participating in consensus on the beacon chain.
func (b *BeaconState) Validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.validators()
}

// validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) validators() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		val := b.state.Validators[i]
		if val == nil {
			continue
		}
		res[i] = copyutil.CopyValidator(val)
	}
	return res
}

// references of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState. This does not
// copy fully and instead just copies the reference.
func (b *BeaconState) validatorsReferences() []*ethpb.Validator {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Validators == nil {
		return nil
	}

	res := make([]*ethpb.Validator, len(b.state.Validators))
	for i := 0; i < len(res); i++ {
		validator := b.state.Validators[i]
		if validator == nil {
			continue
		}
		// copy validator reference instead.
		res[i] = validator
	}
	return res
}

// ValidatorAtIndex is the validator at the provided index.
func (b *BeaconState) ValidatorAtIndex(idx types.ValidatorIndex) (*ethpb.Validator, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return &ethpb.Validator{}, nil
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		e := NewValidatorIndexOutOfRangeError(idx)
		return nil, &e
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	val := b.state.Validators[idx]
	return copyutil.CopyValidator(val), nil
}

// ValidatorAtIndexReadOnly is the validator at the provided index. This method
// doesn't clone the validator.
func (b *BeaconState) ValidatorAtIndexReadOnly(idx types.ValidatorIndex) (iface.ReadOnlyValidator, error) {
	if !b.hasInnerState() {
		return nil, ErrNilInnerState
	}
	if b.state.Validators == nil {
		return nil, ErrNilValidatorsInState
	}
	if uint64(len(b.state.Validators)) <= uint64(idx) {
		e := NewValidatorIndexOutOfRangeError(idx)
		return nil, &e
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return NewValidator(b.state.Validators[idx])
}

// ValidatorIndexByPubkey returns a given validator by its 48-byte public key.
func (b *BeaconState) ValidatorIndexByPubkey(key [48]byte) (types.ValidatorIndex, bool) {
	if b == nil || b.valMapHandler == nil || b.valMapHandler.IsNil() {
		return 0, false
	}
	b.lock.RLock()
	defer b.lock.RUnlock()
	numOfVals := len(b.state.Validators)

	idx, ok := b.valMapHandler.Get(key)
	if ok && numOfVals <= int(idx) {
		return types.ValidatorIndex(0), false
	}
	return idx, ok
}

// PubkeyAtIndex returns the pubkey at the given
// validator index.
func (b *BeaconState) PubkeyAtIndex(idx types.ValidatorIndex) [48]byte {
	if !b.hasInnerState() {
		return [48]byte{}
	}
	if uint64(idx) >= uint64(len(b.state.Validators)) {
		return [48]byte{}
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	if b.state.Validators[idx] == nil {
		return [48]byte{}
	}
	return bytesutil.ToBytes48(b.state.Validators[idx].PublicKey)
}

// NumValidators returns the size of the validator registry.
func (b *BeaconState) NumValidators() int {
	if !b.hasInnerState() {
		return 0
	}
	b.lock.RLock()
	defer b.lock.RUnlock()

	return len(b.state.Validators)
}

// ReadFromEveryValidator reads values from every validator and applies it to the provided function.
// Warning: This method is potentially unsafe, as it exposes the actual validator registry.
func (b *BeaconState) ReadFromEveryValidator(f func(idx int, val iface.ReadOnlyValidator) error) error {
	if !b.hasInnerState() {
		return ErrNilInnerState
	}
	if b.state.Validators == nil {
		return errors.New("nil validators in state")
	}
	b.lock.RLock()
	validators := b.state.Validators
	b.lock.RUnlock()

	for i, v := range validators {
		v, err := NewValidator(v)
		if err != nil {
			return err
		}
		if err := f(i, v); err != nil {
			return err
		}
	}
	return nil
}

// Balances of validators participating in consensus on the beacon chain.
func (b *BeaconState) Balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balances()
}

// balances of validators participating in consensus on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) balances() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Balances == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Balances))
	copy(res, b.state.Balances)
	return res
}

// BalanceAtIndex of validator with the provided index.
func (b *BeaconState) BalanceAtIndex(idx types.ValidatorIndex) (uint64, error) {
	if !b.hasInnerState() {
		return 0, ErrNilInnerState
	}
	if b.state.Balances == nil {
		return 0, nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	if uint64(len(b.state.Balances)) <= uint64(idx) {
		return 0, fmt.Errorf("index of %d does not exist", idx)
	}
	return b.state.Balances[idx], nil
}

// BalancesLength returns the length of the balances slice.
func (b *BeaconState) BalancesLength() int {
	if !b.hasInnerState() {
		return 0
	}
	if b.state.Balances == nil {
		return 0
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.balancesLength()
}

// Slashings of validators on the beacon chain.
func (b *BeaconState) Slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.slashings()
}

// slashings of validators on the beacon chain.
// This assumes that a lock is already held on BeaconState.
func (b *BeaconState) slashings() []uint64 {
	if !b.hasInnerState() {
		return nil
	}
	if b.state.Slashings == nil {
		return nil
	}

	res := make([]uint64, len(b.state.Slashings))
	copy(res, b.state.Slashings)
	return res
}

func (h *stateRootHasher) validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	hashKeyElements := make([]byte, len(validators)*32)
	roots := make([][32]byte, len(validators))
	emptyKey := hashutil.FastSum256(hashKeyElements)
	hasher := hashutil.CustomSHA256Hasher()
	bytesProcessed := 0
	for i := 0; i < len(validators); i++ {
		val, err := h.validatorRoot(hasher, validators[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		copy(hashKeyElements[bytesProcessed:bytesProcessed+32], val[:])
		roots[i] = val
		bytesProcessed += 32
	}

	hashKey := hashutil.FastSum256(hashKeyElements)
	if hashKey != emptyKey && h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(string(hashKey[:])); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	validatorsRootsRoot, err := htrutils.BitwiseMerkleizeArrays(hasher, roots, uint64(len(roots)), params.BeaconConfig().ValidatorRegistryLimit)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute validator registry merkleization")
	}
	validatorsRootsBuf := new(bytes.Buffer)
	if err := binary.Write(validatorsRootsBuf, binary.LittleEndian, uint64(len(validators))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal validator registry length")
	}
	// We need to mix in the length of the slice.
	var validatorsRootsBufRoot [32]byte
	copy(validatorsRootsBufRoot[:], validatorsRootsBuf.Bytes())
	res := htrutils.MixInLength(validatorsRootsRoot, validatorsRootsBufRoot[:])
	if hashKey != emptyKey && h.rootsCache != nil {
		h.rootsCache.Set(string(hashKey[:]), res, 32)
	}
	return res, nil
}

func (h *stateRootHasher) validatorRoot(hasher htrutils.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	if validator == nil {
		return [32]byte{}, errors.New("nil validator")
	}

	enc := stateutil.ValidatorEncKey(validator)
	// Check if it exists in cache:
	if h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	valRoot, err := stateutil.ValidatorRootWithHasher(hasher, validator)
	if err != nil {
		return [32]byte{}, err
	}

	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), valRoot, 32)
	}
	return valRoot, nil
}

// ValidatorRegistryRoot computes the HashTreeRoot Merkleization of
// a list of validator structs according to the Ethereum
// Simple Serialize specification.
func ValidatorRegistryRoot(vals []*ethpb.Validator) ([32]byte, error) {
	if featureconfig.Get().EnableSSZCache {
		return cachedHasher.validatorRegistryRoot(vals)
	}
	return nocachedHasher.validatorRegistryRoot(vals)
}
