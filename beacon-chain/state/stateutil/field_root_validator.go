package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

const (
	// number of field roots for the validator object.
	validatorFieldRoots = 8

	// Depth of tree representation of an individual
	// validator.
	// NumOfRoots = 2 ^ (TreeDepth)
	// 8 = 2 ^ 3
	validatorTreeDepth = 3
)

// ValidatorRegistryRoot computes the HashTreeRoot Merkleization of
// a list of validator structs according to the Ethereum
// Simple Serialize specification.
func ValidatorRegistryRoot(vals []*ethpb.Validator) ([32]byte, error) {
	if features.Get().EnableSSZCache {
		return CachedHasher.validatorRegistryRoot(vals)
	}
	return NocachedHasher.validatorRegistryRoot(vals)
}

func (h *stateRootHasher) validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	hasher := hash.CustomSHA256Hasher()

	var err error
	var roots [][32]byte
	if features.Get().EnableVectorizedHTR {
		roots, err = h.optimizedValidatorRoots(validators)
		if err != nil {
			return [32]byte{}, err
		}
	} else {
		roots, err = h.validatorRoots(hasher, validators)
		if err != nil {
			return [32]byte{}, err
		}
	}

	validatorsRootsRoot, err := ssz.BitwiseMerkleizeArrays(hasher, roots, uint64(len(roots)), fieldparams.ValidatorRegistryLimit)
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
	res := ssz.MixInLength(validatorsRootsRoot, validatorsRootsBufRoot[:])
	return res, nil
}

func (h *stateRootHasher) validatorRoots(hasher func([]byte) [32]byte, validators []*ethpb.Validator) ([][32]byte, error) {
	roots := make([][32]byte, len(validators))
	for i := 0; i < len(validators); i++ {
		val, err := h.validatorRoot(hasher, validators[i])
		if err != nil {
			return [][32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		roots[i] = val
	}
	return roots, nil
}

func (h *stateRootHasher) optimizedValidatorRoots(validators []*ethpb.Validator) ([][32]byte, error) {
	roots := make([][32]byte, 0, len(validators)*validatorFieldRoots)
	hasher := hash.CustomSHA256Hasher()
	for i := 0; i < len(validators); i++ {
		fRoots, err := ValidatorFieldRoots(hasher, validators[i])
		if err != nil {
			return [][32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		roots = append(roots, fRoots...)
	}

	// A validator's tree can represented with a depth of 3. As log2(8) = 3
	// Using this property we can lay out all the individual fields of a
	// validator and hash them in single level using our vectorized routine.
	for i := 0; i < validatorTreeDepth; i++ {
		roots = htr.VectorizedSha256(roots)
	}
	return roots, nil
}

func (h *stateRootHasher) validatorRoot(hasher ssz.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	if validator == nil {
		return [32]byte{}, errors.New("nil validator")
	}

	enc := validatorEncKey(validator)
	// Check if it exists in cache:
	if h.rootsCache != nil {
		if found, ok := h.rootsCache.Get(string(enc)); found != nil && ok {
			return found.([32]byte), nil
		}
	}

	valRoot, err := ValidatorRootWithHasher(hasher, validator)
	if err != nil {
		return [32]byte{}, err
	}

	if h.rootsCache != nil {
		h.rootsCache.Set(string(enc), valRoot, 32)
	}
	return valRoot, nil
}
