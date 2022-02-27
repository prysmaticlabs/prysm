package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/crypto/hash"
	"github.com/prysmaticlabs/prysm/encoding/ssz"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// ValidatorRegistryRoot computes the HashTreeRoot Merkleization of
// a list of validator structs according to the Ethereum
// Simple Serialize specification.
func ValidatorRegistryRoot(vals []*ethpb.Validator) ([32]byte, error) {
	return validatorRegistryRoot(vals)
}

func validatorRegistryRoot(validators []*ethpb.Validator) ([32]byte, error) {
	roots := make([][32]byte, len(validators))
	hasher := hash.CustomSHA256Hasher()
	for i := 0; i < len(validators); i++ {
		val, err := validatorRoot(hasher, validators[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute validators merkleization")
		}
		roots[i] = val
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

func validatorRoot(hasher ssz.HashFn, validator *ethpb.Validator) ([32]byte, error) {
	if validator == nil {
		return [32]byte{}, errors.New("nil validator")
	}
	return ValidatorRootWithHasher(hasher, validator)
}
