package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v4/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
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
	return validatorRegistryRoot(vals)
}

func validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	roots, err := OptimizedValidatorRoots(validators)
	if err != nil {
		return [32]byte{}, err
	}

	validatorsRootsRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), fieldparams.ValidatorRegistryLimit)
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

// OptimizedValidatorRoots uses an optimized routine with gohashtree in order to
// derive a list of validator roots from a list of validator objects.
func OptimizedValidatorRoots(validators []*ethpb.Validator) ([][32]byte, error) {
	// Exit early if no validators are provided.
	if len(validators) == 0 {
		return [][32]byte{}, nil
	}
	roots := make([][32]byte, 0, len(validators)*validatorFieldRoots)
	for i := 0; i < len(validators); i++ {
		fRoots, err := ValidatorFieldRoots(validators[i])
		if err != nil {
			return [][32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		roots = append(roots, fRoots...)
	}

	// A validator's tree can represented with a depth of 3. As log2(8) = 3
	// Using this property we can lay out all the individual fields of a
	// validator and hash them in single level using our vectorized routine.
	for i := 0; i < validatorTreeDepth; i++ {
		// Overwrite input lists as we are hashing by level
		// and only need the highest level to proceed.
		outputLen := len(roots) / 2
		htr.VectorizedSha256(roots, roots)
		roots = roots[:outputLen]
	}
	return roots, nil
}
